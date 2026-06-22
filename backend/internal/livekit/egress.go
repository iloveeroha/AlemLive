package livekit

import (
	"context"
	"errors"
	"fmt"
	"path"
	"regexp"
	"strings"
	"sync"
	"time"

	lkproto "github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
)

var ErrEgressNotConfigured = errors.New("livekit egress is not configured")

type EgressConfig struct {
	Enabled       bool
	ServerURL     string
	APIKey        string
	APISecret     string
	AudioOnly     bool
	Layout        string
	FilePrefix    string
	PublicBaseURL string
	WebhookURL    string
	S3            S3Config
}

type S3Config struct {
	AccessKey      string
	Secret         string
	SessionToken   string
	Region         string
	Endpoint       string
	Bucket         string
	ForcePathStyle bool
}

type EgressManager struct {
	cfg    EgressConfig
	client *lksdk.EgressClient

	mu      sync.Mutex
	active  map[string]EgressState
	history map[string]EgressState
}

type EgressState struct {
	RoomName     string `json:"roomName"`
	EgressID     string `json:"egressId,omitempty"`
	Status       string `json:"status"`
	FilePath     string `json:"filePath,omitempty"`
	FileLocation string `json:"fileLocation,omitempty"`
	PublicURL    string `json:"publicUrl,omitempty"`
	StartedAt    string `json:"startedAt,omitempty"`
	EndedAt      string `json:"endedAt,omitempty"`
	Error        string `json:"error,omitempty"`
}

func NewEgressManager(cfg EgressConfig) *EgressManager {
	manager := &EgressManager{
		cfg:     cfg,
		active:  map[string]EgressState{},
		history: map[string]EgressState{},
	}
	if manager.Configured() {
		manager.client = lksdk.NewEgressClient(cfg.ServerURL, cfg.APIKey, cfg.APISecret)
	}
	return manager
}

func (m *EgressManager) Configured() bool {
	return m != nil &&
		m.cfg.Enabled &&
		strings.TrimSpace(m.cfg.ServerURL) != "" &&
		strings.TrimSpace(m.cfg.APIKey) != "" &&
		strings.TrimSpace(m.cfg.APISecret) != "" &&
		strings.TrimSpace(m.cfg.S3.Bucket) != "" &&
		strings.TrimSpace(m.cfg.S3.Region) != ""
}

func (m *EgressManager) StartRoomComposite(ctx context.Context, roomName string, now time.Time) (EgressState, error) {
	if !m.Configured() || m.client == nil {
		return EgressState{}, ErrEgressNotConfigured
	}
	roomName = strings.TrimSpace(roomName)
	if roomName == "" {
		return EgressState{}, errors.New("roomName is required")
	}

	m.mu.Lock()
	if state, ok := m.active[roomName]; ok {
		m.mu.Unlock()
		return state, nil
	}
	m.mu.Unlock()

	filePath := m.recordingPath(roomName, now)
	req := &lkproto.RoomCompositeEgressRequest{
		RoomName:  roomName,
		Layout:    firstNonEmptyEgress(m.cfg.Layout, "grid"),
		AudioOnly: m.cfg.AudioOnly,
		FileOutputs: []*lkproto.EncodedFileOutput{
			{
				FileType: m.fileType(),
				Filepath: filePath,
				Output: &lkproto.EncodedFileOutput_S3{
					S3: &lkproto.S3Upload{
						AccessKey:      m.cfg.S3.AccessKey,
						Secret:         m.cfg.S3.Secret,
						SessionToken:   m.cfg.S3.SessionToken,
						Region:         m.cfg.S3.Region,
						Endpoint:       m.cfg.S3.Endpoint,
						Bucket:         m.cfg.S3.Bucket,
						ForcePathStyle: m.cfg.S3.ForcePathStyle,
						Metadata: map[string]string{
							"room": roomName,
							"app":  "alemlive",
						},
					},
				},
			},
		},
	}
	if m.cfg.WebhookURL != "" {
		req.Webhooks = []*lkproto.WebhookConfig{{Url: m.cfg.WebhookURL}}
	}

	info, err := m.client.StartRoomCompositeEgress(ctx, req)
	if err != nil {
		return EgressState{}, err
	}

	state := stateFromEgressInfo(info, roomName, filePath, m.publicURL(filePath))
	m.mu.Lock()
	m.active[roomName] = state
	m.history[roomName] = state
	m.mu.Unlock()
	return state, nil
}

func (m *EgressManager) StopRoom(ctx context.Context, roomName string) (EgressState, error) {
	if !m.Configured() || m.client == nil {
		return EgressState{}, ErrEgressNotConfigured
	}

	m.mu.Lock()
	state, ok := m.active[roomName]
	m.mu.Unlock()
	if !ok || state.EgressID == "" {
		return EgressState{}, errors.New("room recording is not active")
	}

	info, err := m.client.StopEgress(ctx, &lkproto.StopEgressRequest{EgressId: state.EgressID})
	if err != nil {
		failed := state
		failed.Status = lkproto.EgressStatus_EGRESS_FAILED.String()
		failed.Error = err.Error()
		failed.EndedAt = time.Now().UTC().Format(time.RFC3339)
		m.mu.Lock()
		delete(m.active, roomName)
		m.history[roomName] = failed
		m.mu.Unlock()
		return EgressState{}, err
	}

	stopped := stateFromEgressInfo(info, roomName, state.FilePath, firstNonEmptyEgress(state.PublicURL, m.publicURL(state.FilePath)))
	m.mu.Lock()
	delete(m.active, roomName)
	m.history[roomName] = stopped
	m.mu.Unlock()
	return stopped, nil
}

func (m *EgressManager) Status(roomName string) (EgressState, bool) {
	if m == nil {
		return EgressState{}, false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if state, ok := m.active[roomName]; ok {
		return state, true
	}
	state, ok := m.history[roomName]
	return state, ok
}

func (m *EgressManager) UpdateFromInfo(info *lkproto.EgressInfo) EgressState {
	if info == nil {
		return EgressState{}
	}
	roomName := info.GetRoomName()
	state := stateFromEgressInfo(info, roomName, "", "")
	if len(info.GetFileResults()) > 0 {
		file := info.GetFileResults()[0]
		state.FilePath = file.GetFilename()
		state.FileLocation = file.GetLocation()
		state.PublicURL = m.publicURL(firstNonEmptyEgress(state.FilePath, state.FileLocation))
	}

	m.mu.Lock()
	if state.RoomName != "" {
		if isEgressTerminal(info.GetStatus()) {
			delete(m.active, state.RoomName)
		} else {
			m.active[state.RoomName] = state
		}
		m.history[state.RoomName] = state
	}
	m.mu.Unlock()
	return state
}

func (m *EgressManager) recordingPath(roomName string, now time.Time) string {
	prefix := sanitizePathPart(firstNonEmptyEgress(m.cfg.FilePrefix, "alemlive"))
	room := sanitizePathPart(roomName)
	ext := "mp4"
	if m.cfg.AudioOnly {
		ext = "ogg"
	}
	return path.Join(prefix, room, fmt.Sprintf("%s.%s", now.UTC().Format("20060102-150405"), ext))
}

func (m *EgressManager) fileType() lkproto.EncodedFileType {
	if m.cfg.AudioOnly {
		return lkproto.EncodedFileType_OGG
	}
	return lkproto.EncodedFileType_MP4
}

func (m *EgressManager) publicURL(filePath string) string {
	filePath = strings.TrimLeft(strings.TrimSpace(filePath), "/")
	if m == nil || m.cfg.PublicBaseURL == "" || filePath == "" || strings.Contains(filePath, "://") {
		return ""
	}
	return strings.TrimRight(m.cfg.PublicBaseURL, "/") + "/" + filePath
}

func stateFromEgressInfo(info *lkproto.EgressInfo, roomName, filePath, publicURL string) EgressState {
	if info == nil {
		return EgressState{RoomName: roomName, Status: "unknown", FilePath: filePath, PublicURL: publicURL}
	}
	if roomName == "" {
		roomName = info.GetRoomName()
	}
	state := EgressState{
		RoomName:  roomName,
		EgressID:  info.GetEgressId(),
		Status:    info.GetStatus().String(),
		FilePath:  filePath,
		PublicURL: publicURL,
		Error:     firstNonEmptyEgress(info.GetError(), info.GetDetails()),
	}
	if info.GetStartedAt() > 0 {
		state.StartedAt = time.Unix(0, info.GetStartedAt()).UTC().Format(time.RFC3339)
	}
	if info.GetEndedAt() > 0 {
		state.EndedAt = time.Unix(0, info.GetEndedAt()).UTC().Format(time.RFC3339)
	}
	if len(info.GetFileResults()) > 0 {
		file := info.GetFileResults()[0]
		state.FilePath = firstNonEmptyEgress(state.FilePath, file.GetFilename())
		state.FileLocation = file.GetLocation()
	}
	return state
}

func isEgressTerminal(status lkproto.EgressStatus) bool {
	return status == lkproto.EgressStatus_EGRESS_COMPLETE ||
		status == lkproto.EgressStatus_EGRESS_FAILED ||
		status == lkproto.EgressStatus_EGRESS_ABORTED ||
		status == lkproto.EgressStatus_EGRESS_LIMIT_REACHED
}

var unsafePathPart = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func sanitizePathPart(value string) string {
	value = strings.Trim(strings.ToLower(value), " ._-")
	value = unsafePathPart.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	if value == "" {
		return "room"
	}
	return value
}

func firstNonEmptyEgress(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
