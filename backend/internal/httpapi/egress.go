package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	lkauth "github.com/livekit/protocol/auth"
	lkproto "github.com/livekit/protocol/livekit"
	lkwebhook "github.com/livekit/protocol/webhook"

	livekitservice "github.com/iloveeroha/AlemLive/backend/internal/livekit"
	"github.com/iloveeroha/AlemLive/backend/internal/llm"
)

func (s *Server) recordingStatus(roomName string) map[string]any {
	response := map[string]any{
		"roomName":   roomName,
		"configured": s.egress != nil && s.egress.Configured(),
		"active":     false,
		"status":     "disabled",
	}
	if s.egress == nil || !s.egress.Configured() {
		return response
	}
	if state, ok := s.egress.Status(roomName); ok {
		response["active"] = state.EgressID != "" && !strings.Contains(strings.ToLower(state.Status), "complete")
		response["status"] = state.Status
		response["recording"] = state
		return response
	}
	response["status"] = "idle"
	return response
}

func (s *Server) startRoomRecording(ctx context.Context, roomName string) (livekitservice.EgressState, error) {
	if s.egress == nil || !s.egress.Configured() {
		return livekitservice.EgressState{}, livekitservice.ErrEgressNotConfigured
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	return s.egress.StartRoomComposite(ctx, roomName, s.clock())
}

func (s *Server) stopRoomRecording(ctx context.Context, roomName string) (livekitservice.EgressState, error) {
	if s.egress == nil || !s.egress.Configured() {
		return livekitservice.EgressState{}, livekitservice.ErrEgressNotConfigured
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	return s.egress.StopRoom(ctx, roomName)
}

func (s *Server) roomRecording(w http.ResponseWriter, r *http.Request, roomName, action string) {
	switch {
	case r.Method == http.MethodGet && action == "":
		writeJSON(w, http.StatusOK, s.recordingStatus(roomName))
	case r.Method == http.MethodPost && (action == "" || action == "start"):
		state, err := s.startRoomRecording(r.Context(), roomName)
		if err != nil {
			writeRecordingError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"roomName": roomName, "status": "recording_started", "recording": state})
	case r.Method == http.MethodPost && action == "stop":
		state, err := s.stopRoomRecording(r.Context(), roomName)
		if err != nil {
			writeRecordingError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"roomName": roomName, "status": "recording_stopped", "recording": state})
	default:
		methodNotAllowed(w, http.MethodGet+", "+http.MethodPost)
	}
}

func writeRecordingError(w http.ResponseWriter, err error) {
	if errors.Is(err, livekitservice.ErrEgressNotConfigured) {
		writeError(w, http.StatusServiceUnavailable, "LiveKit Egress is not configured")
		return
	}
	writeError(w, http.StatusBadGateway, "LiveKit Egress request failed")
}

func (s *Server) liveKitWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	if s.cfg.LiveKitAPIKey == "" || s.cfg.LiveKitSecret == "" {
		writeError(w, http.StatusServiceUnavailable, "LiveKit webhook verification is not configured")
		return
	}

	provider := lkauth.NewSimpleKeyProvider(s.cfg.LiveKitAPIKey, s.cfg.LiveKitSecret)
	event, err := lkwebhook.ReceiveWebhookEvent(r, provider)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Invalid LiveKit webhook")
		return
	}

	response := map[string]any{
		"event":  event.GetEvent(),
		"status": "accepted",
	}
	if info := event.GetEgressInfo(); info != nil && s.egress != nil {
		state := s.egress.UpdateFromInfo(info)
		response["recording"] = state
		if isWebhookEgressTerminal(info) {
			go s.processEgressRecording(context.Background(), state, info)
		}
	}

	writeJSON(w, http.StatusOK, response)
}

func isWebhookEgressTerminal(info *lkproto.EgressInfo) bool {
	if info == nil {
		return false
	}
	return info.GetStatus() == lkproto.EgressStatus_EGRESS_COMPLETE ||
		info.GetStatus() == lkproto.EgressStatus_EGRESS_FAILED ||
		info.GetStatus() == lkproto.EgressStatus_EGRESS_ABORTED ||
		info.GetStatus() == lkproto.EgressStatus_EGRESS_LIMIT_REACHED
}

func (s *Server) processEgressRecording(ctx context.Context, state livekitservice.EgressState, info *lkproto.EgressInfo) {
	if s.stt == nil || !s.stt.Configured() {
		return
	}
	recordingURL := firstRecordingURL(state, info)
	if recordingURL == "" {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	fileName, contentType, data, err := downloadRecording(ctx, recordingURL)
	if err != nil {
		return
	}
	transcription, err := s.stt.Transcribe(ctx, fileName, contentType, data, llm.TranscriptionOptions{
		Model:          s.cfg.STTModel,
		Language:       "ru",
		ResponseFormat: "verbose_json",
	})
	if err != nil {
		return
	}
	transcriptText := truncateRunes(strings.TrimSpace(transcription.Text), maxTranscriptRunes)
	lines := transcriptLinesFromTranscription(transcription)
	if len(lines) == 0 {
		lines = transcriptLinesFromText(transcriptText)
	}
	roomName := firstNonEmpty(state.RoomName, info.GetRoomName(), "alem-meeting")
	analysis, err := s.generateMeetingAnalysisFromTranscript(ctx, roomName, "", transcriptText, lines)
	if err != nil {
		analysis = fallbackAnalysisFromTranscript(roomName, transcriptText, lines, s.clock())
	}

	now := s.clock().UTC()
	reportID := s.latestReportIDForRoom(roomName)
	report := reportRow{
		ID:               firstNonEmpty(reportID, "egress-"+sanitizeReportID(firstNonEmpty(state.EgressID, now.Format("20060102150405")))),
		Title:            "Recording - " + roomName,
		Source:           "LiveKit",
		Type:             "recording",
		Date:             now.Format("02.01.2006"),
		Time:             now.Format("15:04"),
		Participants:     0,
		ParticipantNames: "LiveKit room",
		Score:            90,
		Folder:           "Recordings",
		Owner:            "AlemLive",
		OwnerInitial:     "A",
		ThumbnailTone:    "blue",
		Week:             "LiveKit recordings",
		Duration:         formatTranscriptTime(float64(max(30, len(lines)*30))),
		Status:           "ready",
		ProcessingState:  "ready",
		CreatedAt:        now.Format(time.RFC3339),
		OccurredAt:       now,
	}
	if existing, ok := s.reportDetailForUpdate(report.ID); ok {
		report = existing.Report
		report.Score = 90
		report.Status = "ready"
		report.ProcessingState = "ready"
		report.Duration = formatTranscriptTime(float64(max(30, len(lines)*30)))
		if report.Title == "" {
			report.Title = "Recording - " + roomName
		}
	}
	detail := reportDetailFromAnalysis(report, analysis)
	detail.RecordingURL = recordingURL
	detail.RoomName = roomName
	s.storeReport(detail)
}

func firstRecordingURL(state livekitservice.EgressState, info *lkproto.EgressInfo) string {
	for _, value := range []string{state.PublicURL, state.FileLocation} {
		if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
			return value
		}
	}
	if info != nil {
		for _, file := range info.GetFileResults() {
			if strings.HasPrefix(file.GetLocation(), "http://") || strings.HasPrefix(file.GetLocation(), "https://") {
				return file.GetLocation()
			}
		}
	}
	return ""
}

func downloadRecording(ctx context.Context, url string) (string, string, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", "", nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", nil, errors.New("recording download failed")
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxRecordingUploadBytes+1))
	if err != nil {
		return "", "", nil, err
	}
	if int64(len(data)) > maxRecordingUploadBytes {
		return "", "", nil, errors.New("recording file is too large")
	}
	fileName := path.Base(req.URL.Path)
	if fileName == "." || fileName == "/" || fileName == "" {
		fileName = "recording.ogg"
	}
	return fileName, resp.Header.Get("Content-Type"), data, nil
}

func sanitizeReportID(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteRune('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return time.Now().UTC().Format("20060102150405")
	}
	return out
}

func decodeEgressWebhookForTest(raw []byte) (*lkproto.WebhookEvent, error) {
	var event lkproto.WebhookEvent
	if err := json.Unmarshal(raw, &event); err != nil {
		return nil, err
	}
	return &event, nil
}
