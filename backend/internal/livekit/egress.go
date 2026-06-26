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
var ErrParticipantTracksUnavailable = errors.New("participant audio tracks are not available")

// minRecordingDuration is the minimum time LiveKit Egress needs to boot its
// compositor and receive a first frame. Calling StopEgress sooner reliably
// aborts the egress with "Start signal not received" and produces no file.
const minRecordingDuration = 15 * time.Second

const (
	RecordingModeRoomComposite       = "room_composite"
	RecordingModeParticipantTracks   = "participant_tracks"
	RecordingModeParticipantFallback = "participant_tracks_fallback"
)

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
	RecordingMode string
	FallbackMode  string
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
	RoomName       string             `json:"roomName"`
	EgressID       string             `json:"egressId,omitempty"`
	Status         string             `json:"status"`
	FilePath       string             `json:"filePath,omitempty"`
	FileLocation   string             `json:"fileLocation,omitempty"`
	PublicURL      string             `json:"publicUrl,omitempty"`
	StartedAt      string             `json:"startedAt,omitempty"`
	EndedAt        string             `json:"endedAt,omitempty"`
	Error          string             `json:"error,omitempty"`
	RecordingMode  string             `json:"recordingMode,omitempty"`
	FallbackReason string             `json:"fallbackReason,omitempty"`
	TrackEgresses  []TrackEgressState `json:"trackEgresses,omitempty"`
}

type ParticipantTrack struct {
	ParticipantID       string
	ParticipantIdentity string
	ParticipantName     string
	TrackID             string
}

type TrackEgressState struct {
	EgressID            string `json:"egressId,omitempty"`
	TrackID             string `json:"trackId,omitempty"`
	ParticipantID       string `json:"participantId,omitempty"`
	ParticipantIdentity string `json:"participantIdentity,omitempty"`
	ParticipantName     string `json:"participantName,omitempty"`
	Status              string `json:"status,omitempty"`
	FilePath            string `json:"filePath,omitempty"`
	FileLocation        string `json:"fileLocation,omitempty"`
	PublicURL           string `json:"publicUrl,omitempty"`
	StartedAt           string `json:"startedAt,omitempty"`
	EndedAt             string `json:"endedAt,omitempty"`
	Error               string `json:"error,omitempty"`
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
	info, err := m.startRoomCompositeEgress(ctx, roomName, filePath)
	if err != nil {
		return EgressState{}, err
	}

	state := stateFromEgressInfo(info, roomName, filePath, m.publicURL(filePath))
	state.RecordingMode = RecordingModeRoomComposite
	if state.StartedAt == "" {
		state.StartedAt = now.UTC().Format(time.RFC3339)
	}
	m.mu.Lock()
	m.active[roomName] = state
	m.history[roomName] = state
	m.mu.Unlock()
	return state, nil
}

func (m *EgressManager) StartParticipantTracks(ctx context.Context, roomName string, tracks []ParticipantTrack, now time.Time) (EgressState, error) {
	if !m.Configured() || m.client == nil {
		return EgressState{}, ErrEgressNotConfigured
	}
	roomName = strings.TrimSpace(roomName)
	if roomName == "" {
		return EgressState{}, errors.New("roomName is required")
	}

	tracks = normalizeParticipantTracks(tracks)
	if len(tracks) == 0 {
		return EgressState{}, ErrParticipantTracksUnavailable
	}

	m.mu.Lock()
	if state, ok := m.active[roomName]; ok {
		m.mu.Unlock()
		return state, nil
	}
	m.mu.Unlock()

	filePath := m.recordingPath(roomName, now)
	info, err := m.startRoomCompositeEgress(ctx, roomName, filePath)
	if err != nil {
		return EgressState{}, err
	}

	state := stateFromEgressInfo(info, roomName, filePath, m.publicURL(filePath))
	state.RecordingMode = RecordingModeParticipantTracks
	if state.StartedAt == "" {
		state.StartedAt = now.UTC().Format(time.RFC3339)
	}

	successfulTracks := 0
	for _, track := range tracks {
		trackFilePath := m.trackRecordingPath(roomName, track, now)
		trackState := TrackEgressState{
			TrackID:             track.TrackID,
			ParticipantID:       track.ParticipantID,
			ParticipantIdentity: firstNonEmptyEgress(track.ParticipantIdentity, track.ParticipantID),
			ParticipantName:     track.ParticipantName,
			FilePath:            trackFilePath,
			PublicURL:           m.publicURL(trackFilePath),
			Status:              "starting",
			StartedAt:           now.UTC().Format(time.RFC3339),
		}
		trackInfo, err := m.startTrackEgress(ctx, roomName, track.TrackID, trackFilePath, map[string]string{
			"room":                 roomName,
			"app":                  "alemlive",
			"participant_id":       track.ParticipantID,
			"participant_identity": firstNonEmptyEgress(track.ParticipantIdentity, track.ParticipantID),
			"participant_name":     track.ParticipantName,
			"track_id":             track.TrackID,
			"kind":                 "participant_audio",
		})
		if err != nil {
			trackState.Status = "failed"
			trackState.Error = err.Error()
			state.TrackEgresses = append(state.TrackEgresses, trackState)
			continue
		}
		trackState = trackStateFromEgressInfo(trackInfo, track, trackFilePath, m.publicURL(trackFilePath))
		if trackState.StartedAt == "" {
			trackState.StartedAt = now.UTC().Format(time.RFC3339)
		}
		state.TrackEgresses = append(state.TrackEgresses, trackState)
		successfulTracks++
	}

	if successfulTracks == 0 {
		state.RecordingMode = RecordingModeParticipantFallback
		state.FallbackReason = "participant audio tracks could not be recorded; room composite recording was started"
	} else if successfulTracks < len(tracks) {
		state.FallbackReason = fmt.Sprintf("%d of %d participant audio tracks could not be recorded", len(tracks)-successfulTracks, len(tracks))
	}

	m.mu.Lock()
	m.active[roomName] = state
	m.history[roomName] = state
	m.mu.Unlock()
	return state, nil
}

func (m *EgressManager) AddParticipantTrack(ctx context.Context, roomName string, track ParticipantTrack, now time.Time) (EgressState, error) {
	if !m.Configured() || m.client == nil {
		return EgressState{}, ErrEgressNotConfigured
	}
	roomName = strings.TrimSpace(roomName)
	tracks := normalizeParticipantTracks([]ParticipantTrack{track})
	if roomName == "" || len(tracks) == 0 {
		return EgressState{}, ErrParticipantTracksUnavailable
	}
	track = tracks[0]

	m.mu.Lock()
	state, ok := m.active[roomName]
	if !ok {
		m.mu.Unlock()
		return EgressState{}, ErrParticipantTracksUnavailable
	}
	if state.RecordingMode != RecordingModeParticipantTracks {
		m.mu.Unlock()
		return state, ErrParticipantTracksUnavailable
	}
	for _, existing := range state.TrackEgresses {
		if existing.TrackID == track.TrackID {
			m.mu.Unlock()
			return state, nil
		}
	}
	m.mu.Unlock()

	trackFilePath := m.trackRecordingPath(roomName, track, now)
	info, err := m.startTrackEgress(ctx, roomName, track.TrackID, trackFilePath, map[string]string{
		"room":                 roomName,
		"app":                  "alemlive",
		"participant_id":       track.ParticipantID,
		"participant_identity": firstNonEmptyEgress(track.ParticipantIdentity, track.ParticipantID),
		"participant_name":     track.ParticipantName,
		"track_id":             track.TrackID,
		"kind":                 "participant_audio",
	})
	if err != nil {
		return state, err
	}
	trackState := trackStateFromEgressInfo(info, track, trackFilePath, m.publicURL(trackFilePath))
	if trackState.StartedAt == "" {
		trackState.StartedAt = now.UTC().Format(time.RFC3339)
	}

	m.mu.Lock()
	state, ok = m.active[roomName]
	if !ok {
		m.mu.Unlock()
		return state, nil
	}
	state.TrackEgresses = upsertTrackEgressState(state.TrackEgresses, trackState)
	m.active[roomName] = state
	m.history[roomName] = state
	m.mu.Unlock()
	return state, nil
}

func (m *EgressManager) startRoomCompositeEgress(ctx context.Context, roomName, filePath string) (*lkproto.EgressInfo, error) {
	req := &lkproto.RoomCompositeEgressRequest{
		RoomName:  roomName,
		Layout:    firstNonEmptyEgress(m.cfg.Layout, "grid"),
		AudioOnly: m.cfg.AudioOnly,
		FileOutputs: []*lkproto.EncodedFileOutput{
			{
				FileType: m.fileType(),
				Filepath: filePath,
				Output: &lkproto.EncodedFileOutput_S3{
					S3: m.s3Upload(map[string]string{
						"room": roomName,
						"app":  "alemlive",
						"kind": "room_composite",
					}),
				},
			},
		},
	}
	if m.cfg.WebhookURL != "" {
		req.Webhooks = []*lkproto.WebhookConfig{{
			Url:        m.cfg.WebhookURL,
			SigningKey: m.cfg.APIKey,
		}}
	}
	return m.client.StartRoomCompositeEgress(ctx, req)
}

func (m *EgressManager) startTrackEgress(ctx context.Context, roomName, trackID, filePath string, metadata map[string]string) (*lkproto.EgressInfo, error) {
	req := &lkproto.TrackEgressRequest{
		RoomName: roomName,
		TrackId:  trackID,
		Output: &lkproto.TrackEgressRequest_File{
			File: &lkproto.DirectFileOutput{
				Filepath: filePath,
				Output: &lkproto.DirectFileOutput_S3{
					S3: m.s3Upload(metadata),
				},
			},
		},
	}
	if m.cfg.WebhookURL != "" {
		req.Webhooks = []*lkproto.WebhookConfig{{
			Url:        m.cfg.WebhookURL,
			SigningKey: m.cfg.APIKey,
		}}
	}
	return m.client.StartTrackEgress(ctx, req)
}

func (m *EgressManager) s3Upload(metadata map[string]string) *lkproto.S3Upload {
	return &lkproto.S3Upload{
		AccessKey:      m.cfg.S3.AccessKey,
		Secret:         m.cfg.S3.Secret,
		SessionToken:   m.cfg.S3.SessionToken,
		Region:         m.cfg.S3.Region,
		Endpoint:       m.cfg.S3.Endpoint,
		Bucket:         m.cfg.S3.Bucket,
		ForcePathStyle: m.cfg.S3.ForcePathStyle,
		Metadata:       metadata,
	}
}

func (m *EgressManager) StopRoom(ctx context.Context, roomName string) (EgressState, error) {
	if !m.Configured() || m.client == nil {
		return EgressState{}, ErrEgressNotConfigured
	}

	m.mu.Lock()
	state, ok := m.active[roomName]
	history, hasHistory := m.history[roomName]
	m.mu.Unlock()
	if !ok || (state.EgressID == "" && len(state.TrackEgresses) == 0) {
		if hasHistory && history.EgressID != "" && isEgressTerminalText(history.Status) {
			return history, nil
		}
		return EgressState{}, errors.New("room recording is not active")
	}

	if err := waitForMinRecordingDuration(ctx, state.StartedAt); err != nil {
		return EgressState{}, err
	}

	stopped := state
	var stopErr error
	if state.EgressID != "" {
		info, err := m.client.StopEgress(ctx, &lkproto.StopEgressRequest{EgressId: state.EgressID})
		if err != nil {
			if terminal, ok := m.latestTerminalState(roomName); ok && isLikelyAlreadyStoppedError(err) {
				stopped = terminal
			} else if isLikelyStopAcceptedError(err) {
				stopped.Status = "stopping"
				stopped.Error = err.Error()
				m.mu.Lock()
				m.active[roomName] = stopped
				m.history[roomName] = stopped
				m.mu.Unlock()
				return stopped, nil
			} else {
				stopErr = err
			}
		} else {
			stopped = stateFromEgressInfo(info, roomName, state.FilePath, firstNonEmptyEgress(state.PublicURL, m.publicURL(state.FilePath)))
			stopped.TrackEgresses = state.TrackEgresses
		}
	}

	stopped.RecordingMode = firstNonEmptyEgress(stopped.RecordingMode, state.RecordingMode, RecordingModeRoomComposite)
	stopped.FallbackReason = firstNonEmptyEgress(stopped.FallbackReason, state.FallbackReason)
	stopped.TrackEgresses = m.stopTrackEgresses(ctx, stopped.TrackEgresses)
	if stopErr != nil {
		return stopped, stopErr
	}
	m.mu.Lock()
	delete(m.active, roomName)
	m.history[roomName] = stopped
	m.mu.Unlock()
	return stopped, nil
}

func (m *EgressManager) stopTrackEgresses(ctx context.Context, tracks []TrackEgressState) []TrackEgressState {
	out := make([]TrackEgressState, len(tracks))
	copy(out, tracks)
	for i := range out {
		track := out[i]
		if track.EgressID == "" || isEgressTerminalText(track.Status) {
			continue
		}
		info, err := m.client.StopEgress(ctx, &lkproto.StopEgressRequest{EgressId: track.EgressID})
		if err != nil {
			if isLikelyAlreadyStoppedError(err) {
				track.Status = firstNonEmptyEgress(track.Status, "stopping")
			} else {
				track.Status = "stopping"
				track.Error = err.Error()
			}
			out[i] = track
			continue
		}
		out[i] = mergeTrackEgressState(track, trackStateFromEgressInfo(info, ParticipantTrack{
			ParticipantID:       track.ParticipantID,
			ParticipantIdentity: track.ParticipantIdentity,
			ParticipantName:     track.ParticipantName,
			TrackID:             track.TrackID,
		}, track.FilePath, firstNonEmptyEgress(track.PublicURL, m.publicURL(track.FilePath))))
	}
	return out
}

func (m *EgressManager) latestTerminalState(roomName string) (EgressState, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	state, ok := m.history[roomName]
	return state, ok && state.EgressID != "" && isEgressTerminalText(state.Status)
}

func isLikelyAlreadyStoppedError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "not found") ||
		strings.Contains(message, "not active") ||
		strings.Contains(message, "already") ||
		strings.Contains(message, "complete")
}

func isLikelyStopAcceptedError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "request timed out") ||
		strings.Contains(message, "deadline_exceeded") ||
		strings.Contains(message, "deadline exceeded") ||
		strings.Contains(message, "context deadline exceeded")
}

func isEgressTerminalText(status string) bool {
	status = strings.ToLower(strings.TrimSpace(status))
	return strings.Contains(status, "complete") ||
		strings.Contains(status, "failed") ||
		strings.Contains(status, "aborted") ||
		strings.Contains(status, "limit")
}

// waitForMinRecordingDuration blocks until minRecordingDuration has elapsed
// since startedAt, so StopEgress is never called before the compositor has
// had a chance to receive its first frame.
func waitForMinRecordingDuration(ctx context.Context, startedAt string) error {
	started, err := time.Parse(time.RFC3339, startedAt)
	if err != nil {
		return nil
	}
	remaining := minRecordingDuration - time.Since(started)
	if remaining <= 0 {
		return nil
	}
	timer := time.NewTimer(remaining)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
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
	if info.GetTrack() != nil {
		return m.updateTrackFromInfo(info)
	}
	roomName := firstNonEmptyEgress(info.GetRoomName(), info.GetRoomComposite().GetRoomName())
	state := stateFromEgressInfo(info, roomName, "", "")
	if len(info.GetFileResults()) > 0 {
		file := info.GetFileResults()[0]
		state.FilePath = file.GetFilename()
		state.FileLocation = file.GetLocation()
		state.PublicURL = m.publicURL(firstNonEmptyEgress(state.FilePath, state.FileLocation))
	}

	m.mu.Lock()
	if state.RoomName != "" {
		if previous, ok := m.active[state.RoomName]; ok {
			state.RecordingMode = firstNonEmptyEgress(state.RecordingMode, previous.RecordingMode)
			state.FallbackReason = firstNonEmptyEgress(state.FallbackReason, previous.FallbackReason)
			state.TrackEgresses = previous.TrackEgresses
		} else if previous, ok := m.history[state.RoomName]; ok {
			state.RecordingMode = firstNonEmptyEgress(state.RecordingMode, previous.RecordingMode)
			state.FallbackReason = firstNonEmptyEgress(state.FallbackReason, previous.FallbackReason)
			state.TrackEgresses = previous.TrackEgresses
		}
		if state.RecordingMode == "" {
			state.RecordingMode = RecordingModeRoomComposite
		}
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

func (m *EgressManager) updateTrackFromInfo(info *lkproto.EgressInfo) EgressState {
	trackReq := info.GetTrack()
	trackID := ""
	roomName := info.GetRoomName()
	if trackReq != nil {
		trackID = trackReq.GetTrackId()
		roomName = firstNonEmptyEgress(roomName, trackReq.GetRoomName())
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	state, ok := m.active[roomName]
	if !ok {
		state, ok = m.history[roomName]
	}
	if !ok {
		for _, candidate := range []map[string]EgressState{m.active, m.history} {
			for _, value := range candidate {
				for _, track := range value.TrackEgresses {
					if track.EgressID == info.GetEgressId() {
						state = value
						roomName = value.RoomName
						ok = true
						break
					}
				}
				if ok {
					break
				}
			}
			if ok {
				break
			}
		}
	}
	if !ok {
		state = stateFromEgressInfo(info, roomName, "", "")
		state.RecordingMode = RecordingModeParticipantTracks
	}
	trackState := trackStateFromEgressInfo(info, ParticipantTrack{TrackID: trackID}, "", "")
	state.TrackEgresses = upsertTrackEgressState(state.TrackEgresses, trackState)
	if state.RecordingMode == "" {
		state.RecordingMode = RecordingModeParticipantTracks
	}
	if roomName != "" {
		state.RoomName = roomName
	}
	if state.Status == "" {
		state.Status = "active"
	}
	if state.RoomName != "" {
		if allEgressesTerminal(state) {
			delete(m.active, state.RoomName)
		} else {
			m.active[state.RoomName] = state
		}
		m.history[state.RoomName] = state
	}
	return state
}

func allEgressesTerminal(state EgressState) bool {
	if state.EgressID != "" && !isEgressTerminalText(state.Status) {
		return false
	}
	for _, track := range state.TrackEgresses {
		if track.EgressID != "" && !isEgressTerminalText(track.Status) {
			return false
		}
	}
	return state.EgressID != "" || len(state.TrackEgresses) > 0
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

func (m *EgressManager) trackRecordingPath(roomName string, track ParticipantTrack, now time.Time) string {
	prefix := sanitizePathPart(firstNonEmptyEgress(m.cfg.FilePrefix, "alemlive"))
	room := sanitizePathPart(roomName)
	participant := sanitizePathPart(firstNonEmptyEgress(track.ParticipantName, track.ParticipantIdentity, track.ParticipantID, "participant"))
	trackID := sanitizePathPart(firstNonEmptyEgress(track.TrackID, "track"))
	return path.Join(prefix, room, now.UTC().Format("20060102-150405"), "audio", fmt.Sprintf("%s-%s.ogg", participant, trackID))
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

func trackStateFromEgressInfo(info *lkproto.EgressInfo, track ParticipantTrack, filePath, publicURL string) TrackEgressState {
	state := TrackEgressState{
		TrackID:             track.TrackID,
		ParticipantID:       track.ParticipantID,
		ParticipantIdentity: firstNonEmptyEgress(track.ParticipantIdentity, track.ParticipantID),
		ParticipantName:     track.ParticipantName,
		FilePath:            filePath,
		PublicURL:           publicURL,
		Status:              "unknown",
	}
	if info == nil {
		return state
	}
	if req := info.GetTrack(); req != nil {
		state.TrackID = firstNonEmptyEgress(state.TrackID, req.GetTrackId())
	}
	state.EgressID = info.GetEgressId()
	state.Status = info.GetStatus().String()
	state.Error = firstNonEmptyEgress(info.GetError(), info.GetDetails())
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

func normalizeParticipantTracks(tracks []ParticipantTrack) []ParticipantTrack {
	seen := map[string]struct{}{}
	out := make([]ParticipantTrack, 0, len(tracks))
	for _, track := range tracks {
		track.TrackID = strings.TrimSpace(track.TrackID)
		if track.TrackID == "" {
			continue
		}
		if _, ok := seen[track.TrackID]; ok {
			continue
		}
		seen[track.TrackID] = struct{}{}
		track.ParticipantID = strings.TrimSpace(track.ParticipantID)
		track.ParticipantIdentity = strings.TrimSpace(track.ParticipantIdentity)
		track.ParticipantName = strings.TrimSpace(track.ParticipantName)
		out = append(out, track)
	}
	return out
}

func upsertTrackEgressState(existing []TrackEgressState, incoming TrackEgressState) []TrackEgressState {
	out := append([]TrackEgressState(nil), existing...)
	for i := range out {
		if sameTrackEgress(out[i], incoming) {
			out[i] = mergeTrackEgressState(out[i], incoming)
			return out
		}
	}
	return append(out, incoming)
}

func sameTrackEgress(left, right TrackEgressState) bool {
	if left.EgressID != "" && right.EgressID != "" {
		return left.EgressID == right.EgressID
	}
	return left.TrackID != "" && right.TrackID != "" && left.TrackID == right.TrackID
}

func mergeTrackEgressState(previous, incoming TrackEgressState) TrackEgressState {
	incoming.EgressID = firstNonEmptyEgress(incoming.EgressID, previous.EgressID)
	incoming.TrackID = firstNonEmptyEgress(incoming.TrackID, previous.TrackID)
	incoming.ParticipantID = firstNonEmptyEgress(incoming.ParticipantID, previous.ParticipantID)
	incoming.ParticipantIdentity = firstNonEmptyEgress(incoming.ParticipantIdentity, previous.ParticipantIdentity)
	incoming.ParticipantName = firstNonEmptyEgress(incoming.ParticipantName, previous.ParticipantName)
	incoming.FilePath = firstNonEmptyEgress(incoming.FilePath, previous.FilePath)
	incoming.FileLocation = firstNonEmptyEgress(incoming.FileLocation, previous.FileLocation)
	incoming.PublicURL = firstNonEmptyEgress(incoming.PublicURL, previous.PublicURL)
	incoming.StartedAt = firstNonEmptyEgress(incoming.StartedAt, previous.StartedAt)
	incoming.EndedAt = firstNonEmptyEgress(incoming.EndedAt, previous.EndedAt)
	incoming.Error = firstNonEmptyEgress(incoming.Error, previous.Error)
	incoming.Status = firstNonEmptyEgress(incoming.Status, previous.Status)
	return incoming
}

func firstNonEmptyEgress(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
