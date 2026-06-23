package httpapi

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/gorilla/websocket"
	"github.com/iloveeroha/AlemLive/backend/internal/livekit"
)

var (
	errRoomNotFound                 = errors.New("room not found")
	errParticipantNotFound          = errors.New("participant not found")
	errUnsupportedParticipantAction = errors.New("unsupported participant action")
)

const (
	roomRecordingIdle       = "idle"
	roomRecordingRecording  = "recording"
	roomRecordingProcessing = "processing"
	roomRecordingReady      = "ready"
	roomRecordingError      = "error"
)

type createRoomRequest struct {
	RoomName             string `json:"roomName"`
	UserID               string `json:"userId"`
	UserName             string `json:"userName"`
	InitialMicEnabled    bool   `json:"initialMicEnabled"`
	InitialCameraEnabled bool   `json:"initialCameraEnabled"`
}

type joinRoomRequest struct {
	RoomName             string `json:"roomName"`
	UserID               string `json:"userId"`
	UserName             string `json:"userName"`
	InitialMicEnabled    *bool  `json:"initialMicEnabled"`
	InitialCameraEnabled *bool  `json:"initialCameraEnabled"`
}

type roomUser struct {
	ID   string
	Name string
}

type roomParticipantState struct {
	ID              string
	Name            string
	MicEnabled      bool
	CameraEnabled   bool
	JoinedAt        time.Time
	LastChangedAt   time.Time
	ControlAction   string
	ControlInFlight bool

	MicMutedDuration    time.Duration
	CameraOffDuration   time.Duration
	MicLastChangedAt    time.Time
	CameraLastChangedAt time.Time
}

// setMicEnabled updates the participant's microphone state, accruing muted
// duration for the time it spent disabled since the last change.
func (p *roomParticipantState) setMicEnabled(enabled bool, now time.Time) {
	if !p.MicEnabled {
		base := p.MicLastChangedAt
		if base.IsZero() {
			base = p.JoinedAt
		}
		p.MicMutedDuration += now.Sub(base)
	}
	p.MicEnabled = enabled
	p.MicLastChangedAt = now
}

// setCameraEnabled updates the participant's camera state, accruing
// off-duration for the time it spent disabled since the last change.
func (p *roomParticipantState) setCameraEnabled(enabled bool, now time.Time) {
	if !p.CameraEnabled {
		base := p.CameraLastChangedAt
		if base.IsZero() {
			base = p.JoinedAt
		}
		p.CameraOffDuration += now.Sub(base)
	}
	p.CameraEnabled = enabled
	p.CameraLastChangedAt = now
}

// finalizeDevicePercentages closes out any currently-open muted/off period up
// to now and returns the share of the session spent muted/off.
func (p *roomParticipantState) finalizeDevicePercentages(now time.Time) mediaPercentages {
	total := now.Sub(p.JoinedAt)
	if total <= 0 {
		return mediaPercentages{}
	}

	micMuted := p.MicMutedDuration
	if !p.MicEnabled {
		base := p.MicLastChangedAt
		if base.IsZero() {
			base = p.JoinedAt
		}
		micMuted += now.Sub(base)
	}

	cameraOff := p.CameraOffDuration
	if !p.CameraEnabled {
		base := p.CameraLastChangedAt
		if base.IsZero() {
			base = p.JoinedAt
		}
		cameraOff += now.Sub(base)
	}

	return mediaPercentages{
		MicMutedPercent:  clampScore(int(micMuted.Nanoseconds() * 100 / total.Nanoseconds())),
		CameraOffPercent: clampScore(int(cameraOff.Nanoseconds() * 100 / total.Nanoseconds())),
	}
}

type roomState struct {
	ID             string
	Name           string
	OwnerID        string
	Participants   map[string]*roomParticipantState
	RecordingState string
	ReportID       string
	Closed         bool
	CreatedAt      time.Time
	UpdatedAt      time.Time

	DeviceStatsArchive map[string]mediaPercentages
}

type roomEventEnvelope struct {
	Type    string         `json:"type"`
	Payload map[string]any `json:"payload"`
}

type roomEventClient struct {
	roomID string
	conn   *websocket.Conn
	send   chan roomEventEnvelope
}

var roomEventUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (s *Server) createRoom(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}

	var req createRoomRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32*1024)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	roomName, err := validateField("roomName", req.RoomName)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	user := s.roomUserFromRequest(r, req.UserID, req.UserName)
	snapshot, joined := s.joinRoomState(roomName, user, true, req.InitialMicEnabled, req.InitialCameraEnabled)
	liveKitRoomReady := s.ensureLiveKitRoom(r.Context(), snapshot.Name)
	if joined != nil {
		s.broadcastRoomEvent(snapshot.ID, roomEventEnvelope{Type: "participant_joined", Payload: map[string]any{"participant": joined}})
	}
	s.broadcastRoomEvent(snapshot.ID, roomEventEnvelope{Type: "owner_changed", Payload: map[string]any{"ownerId": snapshot.OwnerID}})

	conference := s.recordConferenceEvent(snapshot.Name, user.Name, "created", s.clock())
	if conference.ReportID != "" {
		snapshot = s.setRoomRecordingState(snapshot.ID, snapshot.RecordingState, conference.ReportID)
	}
	response := s.roomSessionResponse(r, snapshot, user)
	response["liveKitRoomReady"] = liveKitRoomReady
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) joinRoom(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}

	var req joinRoomRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32*1024)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	roomName, err := validateField("roomName", req.RoomName)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	micEnabled := true
	if req.InitialMicEnabled != nil {
		micEnabled = *req.InitialMicEnabled
	}
	cameraEnabled := true
	if req.InitialCameraEnabled != nil {
		cameraEnabled = *req.InitialCameraEnabled
	}

	user := s.roomUserFromRequest(r, req.UserID, req.UserName)
	snapshot, joined := s.joinRoomState(roomName, user, false, micEnabled, cameraEnabled)
	liveKitRoomReady := s.ensureLiveKitRoom(r.Context(), snapshot.Name)
	if joined != nil {
		s.broadcastRoomEvent(snapshot.ID, roomEventEnvelope{Type: "participant_joined", Payload: map[string]any{"participant": joined}})
	}

	conference := s.recordConferenceEvent(snapshot.Name, user.Name, "joined", s.clock())
	if conference.ReportID != "" {
		snapshot = s.setRoomRecordingState(snapshot.ID, snapshot.RecordingState, conference.ReportID)
	}
	response := s.roomSessionResponse(r, snapshot, user)
	response["liveKitRoomReady"] = liveKitRoomReady
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) roomInfo(w http.ResponseWriter, r *http.Request, roomID string) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}

	user := s.roomUserFromRequest(r, "", "")
	snapshot := s.roomSnapshot(roomID)
	writeJSON(w, http.StatusOK, s.roomSessionResponse(r, snapshot, user))
}

func (s *Server) leaveRoom(w http.ResponseWriter, r *http.Request, roomID string) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}

	var req meetingEventRequest
	if r.Body != nil {
		_ = json.NewDecoder(http.MaxBytesReader(w, r.Body, 32*1024)).Decode(&req)
	}

	user := s.roomUserFromRequest(r, "", req.UserName)
	snapshot, leftParticipant, ownerChanged, closed := s.leaveRoomState(roomID, user.ID)
	if leftParticipant != nil {
		s.broadcastRoomEvent(snapshot.ID, roomEventEnvelope{
			Type: "participant_left",
			Payload: map[string]any{
				"participantId": leftParticipant.ID,
				"name":          leftParticipant.Name,
			},
		})
	}
	if ownerChanged {
		s.broadcastRoomEvent(snapshot.ID, roomEventEnvelope{Type: "owner_changed", Payload: map[string]any{"ownerId": snapshot.OwnerID}})
	}
	if closed {
		s.broadcastRoomEvent(snapshot.ID, roomEventEnvelope{Type: "room_closed", Payload: map[string]any{"roomId": snapshot.ID}})
	}

	conference := s.recordConferenceEvent(snapshot.Name, user.Name, "left", s.clock())
	response := s.roomSessionResponse(r, snapshot, user)
	response["status"] = "left"
	response["roomClosed"] = closed
	if conference.ConferenceSaved {
		response["conference"] = conference
		response["reportId"] = conference.ReportID
	}
	if conference.RecordingShouldStop {
		if state, err := s.stopRoomRecording(r.Context(), snapshot.Name); err == nil {
			response["recording"] = state
			response["recordingState"] = roomRecordingProcessing
			s.setRoomRecordingState(snapshot.ID, roomRecordingProcessing, conference.ReportID)
			s.setConferenceReportPipelineState(conference.ReportID, snapshot.Name, roomRecordingProcessing)
			s.scheduleEgressRecordingProcessing(state, nil, 2*time.Second)
			s.broadcastRoomEvent(snapshot.ID, roomEventEnvelope{
				Type:    "recording_stopped",
				Payload: map[string]any{"state": roomRecordingProcessing, "reportId": conference.ReportID},
			})
		}
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) roomDeviceState(w http.ResponseWriter, r *http.Request, roomName string) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}

	var req struct {
		Device  string `json:"device"`
		Enabled bool   `json:"enabled"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4*1024))
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	device := strings.ToLower(strings.TrimSpace(req.Device))
	if device != "mic" && device != "camera" {
		writeError(w, http.StatusBadRequest, "device must be mic or camera")
		return
	}

	user := s.roomUserFromRequest(r, "", "")
	if err := s.applySelfDeviceState(roomName, user.ID, device, req.Enabled); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) roomParticipants(w http.ResponseWriter, r *http.Request, roomID string, parts []string) {
	if len(parts) == 0 {
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}
		user := s.roomUserFromRequest(r, "", "")
		snapshot := s.roomSnapshot(roomID)
		writeJSON(w, http.StatusOK, map[string]any{
			"roomId":       snapshot.ID,
			"participants": snapshot.participantsPayload(user.ID),
		})
		return
	}

	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	if len(parts) < 2 {
		writeError(w, http.StatusNotFound, "Participant action not found")
		return
	}

	participantID := strings.TrimSpace(parts[0])
	action := strings.TrimSpace(parts[1])
	snapshot, participant, eventType, err := s.applyParticipantControl(roomID, participantID, action)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	payload := map[string]any{
		"roomId":        snapshot.ID,
		"participantId": participant.ID,
		"participant":   participant.payload(participant.ID == s.roomUserFromRequest(r, "", "").ID, participant.ID == snapshot.OwnerID),
		"action":        action,
	}
	if eventType != "" {
		s.broadcastRoomEvent(snapshot.ID, roomEventEnvelope{Type: eventType, Payload: payload})
	}
	writeJSON(w, http.StatusOK, payload)
}

func (s *Server) roomEvents(w http.ResponseWriter, r *http.Request, roomID string) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}

	conn, err := roomEventUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	snapshot := s.roomSnapshot(roomID)
	client := &roomEventClient{
		roomID: snapshot.ID,
		conn:   conn,
		send:   make(chan roomEventEnvelope, 16),
	}
	s.registerRoomEventClient(client)
	defer s.unregisterRoomEventClient(client)

	client.send <- roomEventEnvelope{
		Type: "owner_changed",
		Payload: map[string]any{
			"ownerId": snapshot.OwnerID,
			"roomId":  snapshot.ID,
		},
	}
	client.send <- roomEventEnvelope{
		Type: "recording_status_changed",
		Payload: map[string]any{
			"roomId":   snapshot.ID,
			"state":    snapshot.RecordingState,
			"reportId": snapshot.ReportID,
		},
	}

	go client.writePump()
	for {
		if _, _, err := conn.NextReader(); err != nil {
			return
		}
	}
}

func (s *Server) joinRoomState(roomName string, user roomUser, owner bool, micEnabled, cameraEnabled bool) (roomStateSnapshot, map[string]any) {
	now := s.clock().UTC()
	roomID := roomIDFromName(roomName)

	s.roomsMu.Lock()
	defer s.roomsMu.Unlock()
	s.ensureRoomMapsLocked()

	room := s.rooms[roomID]
	if room == nil || room.Closed {
		room = &roomState{
			ID:             roomID,
			Name:           roomName,
			Participants:   map[string]*roomParticipantState{},
			RecordingState: roomRecordingIdle,
			CreatedAt:      now,
		}
		s.rooms[roomID] = room
	}
	room.Name = firstNonEmpty(room.Name, roomName)
	room.UpdatedAt = now
	room.Closed = false
	if room.OwnerID == "" || owner {
		room.OwnerID = user.ID
	}

	participant, existed := room.Participants[user.ID]
	if participant == nil {
		participant = &roomParticipantState{
			ID:       user.ID,
			Name:     user.Name,
			JoinedAt: now,
		}
		room.Participants[user.ID] = participant
	}
	participant.Name = firstNonEmpty(user.Name, participant.Name, user.ID)
	participant.setMicEnabled(micEnabled, now)
	participant.setCameraEnabled(cameraEnabled, now)
	participant.LastChangedAt = now

	snapshot := room.snapshot()
	if existed {
		return snapshot, nil
	}
	return snapshot, participant.payload(true, participant.ID == room.OwnerID)
}

func (s *Server) leaveRoomState(roomID string, participantID string) (roomStateSnapshot, *roomParticipantState, bool, bool) {
	now := s.clock().UTC()
	roomID = strings.TrimSpace(roomID)

	s.roomsMu.Lock()
	defer s.roomsMu.Unlock()
	s.ensureRoomMapsLocked()

	room := s.rooms[roomID]
	if room == nil {
		room = &roomState{
			ID:             roomID,
			Name:           roomID,
			Participants:   map[string]*roomParticipantState{},
			RecordingState: roomRecordingIdle,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		s.rooms[roomID] = room
	}

	left := room.Participants[participantID]
	if left != nil {
		if room.DeviceStatsArchive == nil {
			room.DeviceStatsArchive = map[string]mediaPercentages{}
		}
		room.DeviceStatsArchive[normalizeParticipantNameKey(left.Name)] = left.finalizeDevicePercentages(now)
	}
	delete(room.Participants, participantID)
	room.UpdatedAt = now

	ownerChanged := false
	if room.OwnerID == participantID {
		room.OwnerID = firstParticipantID(room.Participants)
		ownerChanged = room.OwnerID != ""
	}

	closed := len(room.Participants) == 0
	if closed {
		room.Closed = true
		room.OwnerID = ""
	}

	return room.snapshot(), left, ownerChanged, closed
}

func (s *Server) roomSnapshot(roomID string) roomStateSnapshot {
	now := s.clock().UTC()
	roomID = strings.TrimSpace(roomID)

	s.roomsMu.Lock()
	defer s.roomsMu.Unlock()
	s.ensureRoomMapsLocked()

	room := s.rooms[roomID]
	if room == nil {
		room = &roomState{
			ID:             roomID,
			Name:           roomID,
			Participants:   map[string]*roomParticipantState{},
			RecordingState: roomRecordingIdle,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		s.rooms[roomID] = room
	}
	return room.snapshot()
}

func (s *Server) applyParticipantControl(roomID, participantID, action string) (roomStateSnapshot, *roomParticipantState, string, error) {
	now := s.clock().UTC()

	s.roomsMu.Lock()
	defer s.roomsMu.Unlock()

	room := s.rooms[roomID]
	if room == nil {
		return roomStateSnapshot{}, nil, "", errRoomNotFound
	}
	participant := room.Participants[participantID]
	if participant == nil {
		return roomStateSnapshot{}, nil, "", errParticipantNotFound
	}

	eventType := ""
	switch action {
	case "mute":
		participant.setMicEnabled(false, now)
		eventType = "participant_mic_changed"
	case "unmute":
		participant.setMicEnabled(true, now)
		eventType = "participant_mic_changed"
	case "camera-off":
		participant.setCameraEnabled(false, now)
		eventType = "participant_camera_changed"
	case "camera-on-request":
		participant.setCameraEnabled(true, now)
		eventType = "participant_camera_changed"
	default:
		return roomStateSnapshot{}, nil, "", errUnsupportedParticipantAction
	}

	participant.ControlAction = action
	participant.LastChangedAt = now
	room.UpdatedAt = now
	return room.snapshot(), participant.clone(), eventType, nil
}

// applySelfDeviceState records a participant's own mic/camera toggle,
// reported by their client during the call, so the session can later
// compute the real share of time spent muted/off.
func (s *Server) applySelfDeviceState(roomName, participantID, device string, enabled bool) error {
	now := s.clock().UTC()
	roomID := roomIDFromName(roomName)

	s.roomsMu.Lock()
	defer s.roomsMu.Unlock()

	room := s.rooms[roomID]
	if room == nil {
		return errRoomNotFound
	}
	participant := room.Participants[participantID]
	if participant == nil {
		return errParticipantNotFound
	}

	switch device {
	case "mic":
		participant.setMicEnabled(enabled, now)
	case "camera":
		participant.setCameraEnabled(enabled, now)
	default:
		return errUnsupportedParticipantAction
	}

	room.UpdatedAt = now
	return nil
}

// roomDevicePercentages merges archived (already left) and currently
// connected participants' mic/camera percentages for a room, keyed by
// normalized participant name so they can be matched against speaker stats.
func (s *Server) roomDevicePercentages(roomName string) map[string]mediaPercentages {
	roomName = strings.TrimSpace(roomName)
	if roomName == "" {
		return nil
	}
	roomID := roomIDFromName(roomName)
	now := s.clock().UTC()

	s.roomsMu.Lock()
	defer s.roomsMu.Unlock()

	room := s.rooms[roomID]
	if room == nil {
		return nil
	}

	result := make(map[string]mediaPercentages, len(room.DeviceStatsArchive)+len(room.Participants))
	for key, value := range room.DeviceStatsArchive {
		result[key] = value
	}
	for _, participant := range room.Participants {
		result[normalizeParticipantNameKey(participant.Name)] = participant.finalizeDevicePercentages(now)
	}
	return result
}

func (s *Server) setRoomRecordingState(roomID, state, reportID string) roomStateSnapshot {
	s.roomsMu.Lock()
	defer s.roomsMu.Unlock()
	s.ensureRoomMapsLocked()

	room := s.rooms[roomID]
	if room == nil {
		room = &roomState{
			ID:           roomID,
			Name:         roomID,
			Participants: map[string]*roomParticipantState{},
			CreatedAt:    s.clock().UTC(),
		}
		s.rooms[roomID] = room
	}
	room.RecordingState = state
	room.ReportID = firstNonEmpty(reportID, room.ReportID)
	room.UpdatedAt = s.clock().UTC()
	return room.snapshot()
}

func (s *Server) roomSessionResponse(r *http.Request, snapshot roomStateSnapshot, user roomUser) map[string]any {
	liveKitURL, liveKitToken := s.roomLiveKitCredentials(r, snapshot, user)
	response := map[string]any{
		"roomId":         snapshot.ID,
		"id":             snapshot.ID,
		"roomName":       snapshot.Name,
		"name":           snapshot.Name,
		"ownerId":        snapshot.OwnerID,
		"isOwner":        snapshot.OwnerID != "" && snapshot.OwnerID == user.ID,
		"liveKitUrl":     liveKitURL,
		"liveKitToken":   liveKitToken,
		"recordingState": snapshot.RecordingState,
		"participants":   snapshot.participantsPayload(user.ID),
	}
	if snapshot.ReportID != "" {
		response["reportId"] = snapshot.ReportID
	}
	return response
}

func (s *Server) roomLiveKitCredentials(r *http.Request, snapshot roomStateSnapshot, user roomUser) (string, string) {
	if s.cfg.LiveKitURL == "" || s.cfg.LiveKitAPIKey == "" || s.cfg.LiveKitSecret == "" {
		return "", ""
	}

	role := "participant"
	if snapshot.OwnerID == user.ID {
		role = "host"
	}
	metadata, err := json.Marshal(map[string]string{"role": role})
	if err != nil {
		return "", ""
	}

	token, _, err := livekit.GenerateToken(
		s.cfg.LiveKitAPIKey,
		s.cfg.LiveKitSecret,
		user.ID,
		snapshot.Name,
		string(metadata),
		s.cfg.TokenTTL,
		s.clock(),
	)
	if err != nil {
		return "", ""
	}
	return s.publicLiveKitURL(r), token
}

func (s *Server) ensureLiveKitRoom(ctx context.Context, roomName string) bool {
	if s.cfg.LiveKitURL == "" || s.cfg.LiveKitAPIKey == "" || s.cfg.LiveKitSecret == "" {
		return false
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return livekit.EnsureRoom(ctx, s.cfg.LiveKitURL, s.cfg.LiveKitAPIKey, s.cfg.LiveKitSecret, roomName) == nil
}

func (s *Server) roomUserFromRequest(r *http.Request, explicitID, explicitName string) roomUser {
	if user, ok := userFromContext(r.Context()); ok {
		return roomUser{
			ID:   firstNonEmpty(explicitID, user.ID, user.Username, user.Email),
			Name: firstNonEmpty(explicitName, user.Name, user.Username, user.Email, user.ID),
		}
	}

	name := firstNonEmpty(explicitName, r.URL.Query().Get("userName"), "Madi")
	id := firstNonEmpty(explicitID, r.URL.Query().Get("userId"), "local-user")
	return roomUser{ID: id, Name: name}
}

func (s *Server) ensureRoomMapsLocked() {
	if s.rooms == nil {
		s.rooms = map[string]*roomState{}
	}
	if s.roomClients == nil {
		s.roomClients = map[string]map[*roomEventClient]struct{}{}
	}
}

func (s *Server) registerRoomEventClient(client *roomEventClient) {
	s.roomsMu.Lock()
	defer s.roomsMu.Unlock()
	s.ensureRoomMapsLocked()

	if s.roomClients[client.roomID] == nil {
		s.roomClients[client.roomID] = map[*roomEventClient]struct{}{}
	}
	s.roomClients[client.roomID][client] = struct{}{}
}

func (s *Server) unregisterRoomEventClient(client *roomEventClient) {
	s.roomsMu.Lock()
	defer s.roomsMu.Unlock()
	if s.roomClients != nil && s.roomClients[client.roomID] != nil {
		delete(s.roomClients[client.roomID], client)
	}
	close(client.send)
	_ = client.conn.Close()
}

func (s *Server) broadcastRoomEvent(roomID string, event roomEventEnvelope) {
	s.roomsMu.Lock()
	defer s.roomsMu.Unlock()
	if s.roomClients == nil {
		return
	}
	for client := range s.roomClients[roomID] {
		select {
		case client.send <- event:
		default:
		}
	}
}

func (client *roomEventClient) writePump() {
	for event := range client.send {
		if err := client.conn.WriteJSON(event); err != nil {
			return
		}
	}
}

type roomStateSnapshot struct {
	ID             string
	Name           string
	OwnerID        string
	Participants   []*roomParticipantState
	RecordingState string
	ReportID       string
	Closed         bool
}

func (room *roomState) snapshot() roomStateSnapshot {
	participants := make([]*roomParticipantState, 0, len(room.Participants))
	for _, participant := range room.Participants {
		participants = append(participants, participant.clone())
	}
	sort.Slice(participants, func(i, j int) bool {
		return participants[i].JoinedAt.Before(participants[j].JoinedAt)
	})
	return roomStateSnapshot{
		ID:             room.ID,
		Name:           room.Name,
		OwnerID:        room.OwnerID,
		Participants:   participants,
		RecordingState: firstNonEmpty(room.RecordingState, roomRecordingIdle),
		ReportID:       room.ReportID,
		Closed:         room.Closed,
	}
}

func (snapshot roomStateSnapshot) participantsPayload(currentUserID string) []map[string]any {
	out := make([]map[string]any, 0, len(snapshot.Participants))
	for _, participant := range snapshot.Participants {
		out = append(out, participant.payload(participant.ID == currentUserID, participant.ID == snapshot.OwnerID))
	}
	return out
}

func (participant *roomParticipantState) payload(currentUser, owner bool) map[string]any {
	return map[string]any{
		"id":              participant.ID,
		"participantId":   participant.ID,
		"name":            participant.Name,
		"isCurrentUser":   currentUser,
		"isOwner":         owner,
		"isMicEnabled":    participant.MicEnabled,
		"isCameraEnabled": participant.CameraEnabled,
		"micEnabled":      participant.MicEnabled,
		"cameraEnabled":   participant.CameraEnabled,
		"controlAction":   participant.ControlAction,
	}
}

func (participant *roomParticipantState) clone() *roomParticipantState {
	if participant == nil {
		return nil
	}
	copy := *participant
	return &copy
}

func firstParticipantID(participants map[string]*roomParticipantState) string {
	ids := make([]string, 0, len(participants))
	for id := range participants {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	if len(ids) == 0 {
		return ""
	}
	return ids[0]
}

func roomIDFromName(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			lastDash = false
		case !lastDash:
			b.WriteRune('-')
			lastDash = true
		}
	}
	id := strings.Trim(b.String(), "-")
	if id != "" && len([]rune(id)) <= 72 {
		return id
	}

	sum := sha1.Sum([]byte(value))
	return "room-" + hex.EncodeToString(sum[:])[:12]
}

func normalizeParticipantNameKey(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
