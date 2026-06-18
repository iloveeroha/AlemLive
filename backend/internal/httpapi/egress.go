package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
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
	snapshot := s.roomSnapshot(roomName)
	liveKitRoomName := firstNonEmpty(snapshot.Name, roomName)

	switch {
	case r.Method == http.MethodGet && (action == "" || action == "status"):
		writeJSON(w, http.StatusOK, s.roomRecordingStatusPayload(snapshot))
	case r.Method == http.MethodPost && (action == "" || action == "start"):
		_ = s.ensureLiveKitRoom(r.Context(), liveKitRoomName)
		state, err := s.startRoomRecording(r.Context(), liveKitRoomName)
		if err != nil {
			recordingState := roomRecordingError
			if errors.Is(err, livekitservice.ErrEgressNotConfigured) {
				recordingState = roomRecordingIdle
			}
			s.setRoomRecordingState(snapshot.ID, recordingState, snapshot.ReportID)
			payload := map[string]any{
				"roomId":     snapshot.ID,
				"roomName":   liveKitRoomName,
				"status":     recordingState,
				"state":      recordingState,
				"configured": s.egress != nil && s.egress.Configured(),
				"error":      recordingErrorMessage(err),
			}
			s.broadcastRoomEvent(snapshot.ID, roomEventEnvelope{Type: "recording_status_changed", Payload: payload})
			writeJSON(w, http.StatusOK, payload)
			return
		}
		s.setRoomRecordingState(snapshot.ID, roomRecordingRecording, snapshot.ReportID)
		payload := map[string]any{
			"roomId":     snapshot.ID,
			"roomName":   liveKitRoomName,
			"status":     roomRecordingRecording,
			"state":      roomRecordingRecording,
			"configured": true,
			"recording":  state,
		}
		s.broadcastRoomEvent(snapshot.ID, roomEventEnvelope{Type: "recording_started", Payload: payload})
		writeJSON(w, http.StatusOK, payload)
	case r.Method == http.MethodPost && action == "stop":
		state, err := s.stopRoomRecording(r.Context(), liveKitRoomName)
		if err != nil {
			recordingState := roomRecordingError
			if errors.Is(err, livekitservice.ErrEgressNotConfigured) {
				recordingState = roomRecordingIdle
			}
			s.setRoomRecordingState(snapshot.ID, recordingState, snapshot.ReportID)
			payload := map[string]any{
				"roomId":     snapshot.ID,
				"roomName":   liveKitRoomName,
				"status":     recordingState,
				"state":      recordingState,
				"configured": s.egress != nil && s.egress.Configured(),
				"error":      recordingErrorMessage(err),
			}
			s.broadcastRoomEvent(snapshot.ID, roomEventEnvelope{Type: "recording_status_changed", Payload: payload})
			writeJSON(w, http.StatusOK, payload)
			return
		}
		s.setRoomRecordingState(snapshot.ID, roomRecordingProcessing, snapshot.ReportID)
		payload := map[string]any{
			"roomId":     snapshot.ID,
			"roomName":   liveKitRoomName,
			"status":     roomRecordingProcessing,
			"state":      roomRecordingProcessing,
			"configured": true,
			"recording":  state,
			"reportId":   snapshot.ReportID,
		}
		s.broadcastRoomEvent(snapshot.ID, roomEventEnvelope{Type: "recording_stopped", Payload: payload})
		writeJSON(w, http.StatusOK, payload)
	default:
		methodNotAllowed(w, http.MethodGet+", "+http.MethodPost)
	}
}

func (s *Server) roomRecordingStatusPayload(snapshot roomStateSnapshot) map[string]any {
	state := firstNonEmpty(snapshot.RecordingState, roomRecordingIdle)
	status := s.recordingStatus(firstNonEmpty(snapshot.Name, snapshot.ID))
	if egressState := normalizeRoomRecordingState(status); egressState != "" {
		if state == roomRecordingIdle || egressState != roomRecordingIdle {
			state = egressState
		}
	}
	s.setRoomRecordingState(snapshot.ID, state, snapshot.ReportID)

	return map[string]any{
		"roomId":     snapshot.ID,
		"roomName":   snapshot.Name,
		"status":     state,
		"state":      state,
		"reportId":   snapshot.ReportID,
		"configured": status["configured"],
		"active":     status["active"],
		"recording":  status["recording"],
	}
}

func normalizeRoomRecordingState(payload map[string]any) string {
	active, _ := payload["active"].(bool)
	statusValue, _ := payload["status"].(string)
	status := strings.ToLower(strings.TrimSpace(statusValue))
	switch {
	case strings.Contains(status, "fail") || strings.Contains(status, "abort") || strings.Contains(status, "error"):
		return roomRecordingError
	case strings.Contains(status, "complete") || strings.Contains(status, "ready"):
		return roomRecordingReady
	case active || strings.Contains(status, "active") || strings.Contains(status, "record"):
		return roomRecordingRecording
	case status == "idle" || status == "disabled" || status == "":
		return roomRecordingIdle
	default:
		return ""
	}
}

func recordingErrorMessage(err error) string {
	if errors.Is(err, livekitservice.ErrEgressNotConfigured) {
		return "LiveKit Egress is not configured"
	}
	return "LiveKit Egress request failed"
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
		roomID := roomIDFromName(firstNonEmpty(state.RoomName, info.GetRoomName()))
		recordingState := normalizeLiveKitEgressStatus(info.GetStatus().String())
		if recordingState == "" {
			recordingState = roomRecordingRecording
		}
		s.setRoomRecordingState(roomID, recordingState, "")
		s.broadcastRoomEvent(roomID, roomEventEnvelope{
			Type: "recording_status_changed",
			Payload: map[string]any{
				"roomId":    roomID,
				"roomName":  firstNonEmpty(state.RoomName, info.GetRoomName()),
				"state":     recordingState,
				"status":    recordingState,
				"recording": state,
			},
		})
		if isWebhookEgressTerminal(info) {
			go s.processEgressRecording(context.Background(), state, info)
		}
	}

	writeJSON(w, http.StatusOK, response)
}

func normalizeLiveKitEgressStatus(status string) string {
	status = strings.ToLower(strings.TrimSpace(status))
	switch {
	case strings.Contains(status, "complete"):
		return roomRecordingProcessing
	case strings.Contains(status, "fail") || strings.Contains(status, "abort") || strings.Contains(status, "limit"):
		return roomRecordingError
	case status != "":
		return roomRecordingRecording
	default:
		return ""
	}
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
	downloadURL := s.firstRecordingDownloadURL(state, info)
	if downloadURL == "" {
		log.Printf("egress report skipped: no recording download URL for room=%s egress=%s", state.RoomName, state.EgressID)
		return
	}
	playbackURL := s.firstRecordingPlaybackURL(state, info)
	roomName := firstNonEmpty(state.RoomName, info.GetRoomName(), "alem-meeting")
	now := s.clock().UTC()
	reportID := firstNonEmpty(
		s.latestReportIDForRoom(roomName),
		"egress-"+sanitizeReportID(firstNonEmpty(state.EgressID, now.Format("20060102150405"))),
	)
	detail := s.upsertEgressRecordingDetail(reportID, roomName, playbackURL, now)

	if s.stt == nil || !s.stt.Configured() {
		detail.Report.Status = "ready"
		detail.Report.ProcessingState = "ready"
		detail.Report.RecordingStatus = "completed"
		detail.Report.TranscriptionStatus = "not_configured"
		detail.Report.AnalysisStatus = "not_started"
		detail.Summary = []summarySection{
			{Title: "Recording captured", Text: "Meeting video is available. Configure STT to generate transcript and AI analysis automatically."},
		}
		s.storeReport(detail)
		s.publishRoomRecordingReady(roomName, reportID)
		return
	}

	ctx, cancel := context.WithTimeout(ctx, s.processingTimeout())
	defer cancel()

	fileName, contentType, data, err := downloadRecording(ctx, downloadURL)
	if err != nil {
		log.Printf("egress report %s recording download failed: %v", reportID, err)
		s.markEgressReportFailed(reportID, err)
		return
	}
	transcription, err := s.stt.Transcribe(ctx, fileName, contentType, data, llm.TranscriptionOptions{
		Model:          s.cfg.STTModel,
		Language:       "ru",
		ResponseFormat: "verbose_json",
	})
	if err != nil {
		log.Printf("egress report %s transcription failed: %v", reportID, err)
		s.markEgressReportFailed(reportID, err)
		return
	}
	transcriptText := truncateRunes(strings.TrimSpace(transcription.Text), maxTranscriptRunes)
	lines := transcriptLinesFromTranscription(transcription)
	if len(lines) == 0 {
		lines = transcriptLinesFromText(transcriptText)
	}
	participants := strings.TrimSpace(detail.Report.ParticipantNames)
	lines = s.diarizeTranscript(ctx, fileName, contentType, data, participants, lines)
	analysis, err := s.generateMeetingAnalysisFromTranscript(ctx, roomName, participants, transcriptText, lines)
	analysisStatus := "completed"
	if err != nil {
		log.Printf("egress report %s meeting analysis fallback: %v", reportID, err)
		analysis = fallbackAnalysisFromTranscript(roomName, transcriptText, lines, s.clock())
		analysisStatus = "failed"
	}

	report := detail.Report
	report.Score = 90
	report.Status = "ready"
	report.ProcessingState = "ready"
	report.RecordingStatus = "completed"
	report.TranscriptionStatus = "completed"
	report.AnalysisStatus = analysisStatus
	report.Duration = formatTranscriptTime(float64(max(30, len(lines)*30)))
	if report.Title == "" {
		report.Title = "Recording - " + roomName
	}
	readyDetail := reportDetailFromAnalysis(report, analysis)
	readyDetail.RecordingURL = firstNonEmpty(playbackURL, downloadURL)
	readyDetail.RoomName = roomName
	s.storeReport(readyDetail)
	s.publishRoomRecordingReady(roomName, reportID)
}

func (s *Server) upsertEgressRecordingDetail(reportID, roomName, playbackURL string, now time.Time) reportDetailResponse {
	if existing, ok := s.reportDetailForUpdate(reportID); ok {
		existing.RecordingURL = firstNonEmpty(playbackURL, existing.RecordingURL)
		existing.RoomName = firstNonEmpty(existing.RoomName, roomName)
		existing.Report.Status = "processing"
		existing.Report.ProcessingState = "processing"
		existing.Report.RecordingStatus = "completed"
		existing.Report.TranscriptionStatus = "pending"
		existing.Report.AnalysisStatus = "pending"
		s.storeReport(existing)
		return existing
	}

	report := reportRow{
		ID:                  reportID,
		Title:               "Recording - " + roomName,
		Source:              "LiveKit",
		Type:                "recording",
		Date:                now.Format("02.01.2006"),
		Time:                now.Format("15:04"),
		Participants:        0,
		ParticipantNames:    "LiveKit room",
		Score:               0,
		Folder:              "Recordings",
		Owner:               "AlemLive",
		OwnerInitial:        "A",
		ThumbnailTone:       "blue",
		Week:                "LiveKit recordings",
		Duration:            "00:00",
		Status:              "processing",
		ProcessingState:     "processing",
		RecordingStatus:     "completed",
		TranscriptionStatus: "pending",
		AnalysisStatus:      "pending",
		CreatedAt:           now.Format(time.RFC3339),
		OccurredAt:          now,
	}
	detail := reportDetailResponse{
		Report: report,
		Summary: []summarySection{
			{Title: "Recording captured", Text: "Meeting video is saved. Backend is generating transcript and AI analysis."},
		},
		ActionItems:     []reportActionItem{},
		TranscriptLines: []reportTranscript{},
		Transcript:      []reportTranscript{},
		SpeakerStats:    []speakerStat{},
		Highlights:      []highlight{},
		Chapters:        []chapter{},
		AIQuestions: []string{
			"What were the key decisions?",
			"What action items appeared after the meeting?",
			"Summarize this meeting in Russian.",
		},
		RecordingURL: playbackURL,
		RoomName:     roomName,
	}
	s.storeReport(detail)
	return detail
}

func (s *Server) markEgressReportFailed(reportID string, err error) {
	detail, ok := s.reportDetailByID(reportID)
	if !ok {
		return
	}
	detail.Report.Status = "ready"
	detail.Report.ProcessingState = "ready"
	if detail.Report.RecordingStatus == "" {
		detail.Report.RecordingStatus = "completed"
	}
	detail.Report.TranscriptionStatus = "failed"
	if detail.Report.AnalysisStatus == "" {
		detail.Report.AnalysisStatus = "not_started"
	}
	detail.Summary = []summarySection{
		{Title: "Recording captured", Text: "Meeting video is available, but transcript/AI processing failed: " + reportProcessingErrorMessage(err)},
	}
	s.storeReport(detail)
	s.publishRoomRecordingReady(detail.RoomName, reportID)
}

func (s *Server) publishRoomRecordingReady(roomName, reportID string) {
	roomID := roomIDFromName(roomName)
	s.setRoomRecordingState(roomID, roomRecordingReady, reportID)
	s.broadcastRoomEvent(roomID, roomEventEnvelope{
		Type: "recording_status_changed",
		Payload: map[string]any{
			"roomId":   roomID,
			"roomName": roomName,
			"state":    roomRecordingReady,
			"status":   roomRecordingReady,
			"reportId": reportID,
		},
	})
}

func (s *Server) processingTimeout() time.Duration {
	timeout := s.cfg.STTTimeout + s.cfg.DiarizationTimeout + s.cfg.LLMTimeout + time.Minute
	if timeout <= time.Minute {
		return 20 * time.Minute
	}
	return timeout
}

func (s *Server) firstRecordingDownloadURL(state livekitservice.EgressState, info *lkproto.EgressInfo) string {
	if url := s.s3ObjectURL(s.cfg.LiveKitS3Endpoint, s.cfg.LiveKitS3Bucket, firstEgressFilePath(state, info)); url != "" {
		return url
	}
	for _, value := range []string{state.FileLocation, state.PublicURL} {
		if isHTTPURL(value) {
			return value
		}
	}
	if info != nil {
		for _, file := range info.GetFileResults() {
			if isHTTPURL(file.GetLocation()) {
				return file.GetLocation()
			}
		}
	}
	return ""
}

func (s *Server) firstRecordingPlaybackURL(state livekitservice.EgressState, info *lkproto.EgressInfo) string {
	for _, value := range []string{state.PublicURL, state.FileLocation} {
		if isHTTPURL(value) {
			return value
		}
	}
	if info != nil {
		for _, file := range info.GetFileResults() {
			if isHTTPURL(file.GetLocation()) {
				return file.GetLocation()
			}
		}
	}
	return s.s3ObjectURL(s.cfg.LiveKitS3Endpoint, s.cfg.LiveKitS3Bucket, firstEgressFilePath(state, info))
}

func (s *Server) s3ObjectURL(baseURL, bucket, filePath string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	bucket = strings.Trim(strings.TrimSpace(bucket), "/")
	filePath = strings.TrimLeft(strings.TrimSpace(filePath), "/")
	if baseURL == "" || bucket == "" || filePath == "" || !isHTTPURL(baseURL) {
		return ""
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	return baseURL + "/" + bucket + "/" + filePath
}

func firstEgressFilePath(state livekitservice.EgressState, info *lkproto.EgressInfo) string {
	if state.FilePath != "" && !isHTTPURL(state.FilePath) {
		return state.FilePath
	}
	if info != nil {
		for _, file := range info.GetFileResults() {
			if file.GetFilename() != "" {
				return file.GetFilename()
			}
		}
	}
	return ""
}

func isHTTPURL(value string) bool {
	value = strings.TrimSpace(value)
	return strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://")
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
