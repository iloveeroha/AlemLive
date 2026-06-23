package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return s.egress.StopRoom(ctx, roomName)
}

func (s *Server) roomRecording(w http.ResponseWriter, r *http.Request, roomName, action string) {
	snapshot := s.roomSnapshot(roomName)
	liveKitRoomName := firstNonEmpty(snapshot.Name, roomName)
	reportID := firstNonEmpty(snapshot.ReportID, s.latestReportIDForRoom(liveKitRoomName))

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
			s.setRoomRecordingState(snapshot.ID, recordingState, reportID)
			if recordingState == roomRecordingError {
				s.setConferenceReportPipelineState(reportID, liveKitRoomName, roomRecordingError)
			}
			payload := map[string]any{
				"roomId":     snapshot.ID,
				"roomName":   liveKitRoomName,
				"status":     recordingState,
				"state":      recordingState,
				"configured": s.egress != nil && s.egress.Configured(),
				"error":      recordingErrorMessage(err),
				"reportId":   reportID,
			}
			s.broadcastRoomEvent(snapshot.ID, roomEventEnvelope{Type: "recording_status_changed", Payload: payload})
			writeJSON(w, http.StatusOK, payload)
			return
		}
		s.setRoomRecordingState(snapshot.ID, roomRecordingRecording, reportID)
		s.setConferenceReportPipelineState(reportID, liveKitRoomName, roomRecordingRecording)
		payload := map[string]any{
			"roomId":     snapshot.ID,
			"roomName":   liveKitRoomName,
			"status":     roomRecordingRecording,
			"state":      roomRecordingRecording,
			"configured": true,
			"recording":  state,
			"reportId":   reportID,
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
			s.setRoomRecordingState(snapshot.ID, recordingState, reportID)
			if recordingState == roomRecordingError {
				s.setConferenceReportPipelineState(reportID, liveKitRoomName, roomRecordingError)
			}
			payload := map[string]any{
				"roomId":     snapshot.ID,
				"roomName":   liveKitRoomName,
				"status":     recordingState,
				"state":      recordingState,
				"configured": s.egress != nil && s.egress.Configured(),
				"error":      recordingErrorMessage(err),
				"reportId":   reportID,
			}
			s.broadcastRoomEvent(snapshot.ID, roomEventEnvelope{Type: "recording_status_changed", Payload: payload})
			writeJSON(w, http.StatusOK, payload)
			return
		}
		s.setRoomRecordingState(snapshot.ID, roomRecordingProcessing, reportID)
		s.setConferenceReportPipelineState(reportID, liveKitRoomName, roomRecordingProcessing)
		s.scheduleEgressRecordingProcessing(state, nil, 2*time.Second)
		payload := map[string]any{
			"roomId":     snapshot.ID,
			"roomName":   liveKitRoomName,
			"status":     roomRecordingProcessing,
			"state":      roomRecordingProcessing,
			"configured": true,
			"recording":  state,
			"reportId":   reportID,
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
		roomName := firstNonEmpty(state.RoomName, info.GetRoomName())
		roomID := roomIDFromName(roomName)
		recordingState := normalizeLiveKitEgressStatus(info.GetStatus().String())
		if recordingState == "" {
			recordingState = roomRecordingRecording
		}
		s.setRoomRecordingState(roomID, recordingState, "")
		if recordingState == roomRecordingProcessing || recordingState == roomRecordingError {
			s.setConferenceReportPipelineState("", roomName, recordingState)
		}
		s.broadcastRoomEvent(roomID, roomEventEnvelope{
			Type: "recording_status_changed",
			Payload: map[string]any{
				"roomId":    roomID,
				"roomName":  roomName,
				"state":     recordingState,
				"status":    recordingState,
				"recording": state,
			},
		})
		if isWebhookEgressTerminal(info) {
			s.scheduleEgressRecordingProcessing(state, info, 0)
		}
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) scheduleEgressRecordingProcessing(state livekitservice.EgressState, info *lkproto.EgressInfo, delay time.Duration) {
	key := firstNonEmpty(state.EgressID, state.FilePath, state.RoomName)
	if key == "" {
		return
	}
	s.egressProcessingMu.Lock()
	if s.egressProcessing == nil {
		s.egressProcessing = map[string]struct{}{}
	}
	if _, exists := s.egressProcessing[key]; exists {
		s.egressProcessingMu.Unlock()
		return
	}
	s.egressProcessing[key] = struct{}{}
	s.egressProcessingMu.Unlock()

	go func() {
		defer func() {
			s.egressProcessingMu.Lock()
			delete(s.egressProcessing, key)
			s.egressProcessingMu.Unlock()
		}()

		if delay > 0 {
			timer := time.NewTimer(delay)
			defer timer.Stop()
			<-timer.C
		}
		s.processEgressRecording(context.Background(), state, info)
	}()
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
	playbackURL := s.firstRecordingPlaybackURL(state, info)
	roomName := firstNonEmpty(state.RoomName, info.GetRoomName(), "alem-meeting")
	now := s.clock().UTC()
	reportID := firstNonEmpty(
		s.latestReportIDForRoom(roomName),
		"egress-"+sanitizeReportID(firstNonEmpty(state.EgressID, now.Format("20060102150405"))),
	)
	downloadURL := s.firstRecordingDownloadURL(state, info)
	if downloadURL == "" {
		log.Printf("egress report %s skipped: no recording download URL for room=%s egress=%s", reportID, state.RoomName, state.EgressID)
		s.setConferenceReportPipelineState(reportID, roomName, roomRecordingError)
		return
	}
	detail := s.upsertEgressRecordingDetail(reportID, roomName, playbackURL, downloadURL, now)

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

	status := strings.ToLower(strings.TrimSpace(state.Status))
	if strings.Contains(status, "abort") || strings.Contains(status, "fail") || strings.Contains(status, "limit") {
		log.Printf("egress report %s aborted before recording completion: status=%s filePath=%s", reportID, state.Status, state.FilePath)
		s.markEgressReportFailed(reportID, ErrRecordingDownloadFailed)
		return
	}

	downloadURLs := s.recordingDownloadURLs(state, info)
	if len(downloadURLs) == 0 {
		log.Printf("egress report %s skipped: no recording download URL for room=%s egress=%s status=%s", reportID, state.RoomName, state.EgressID, state.Status)
		s.setConferenceReportPipelineState(reportID, roomName, roomRecordingError)
		return
	}

	fileName, contentType, data, downloadURL, err := downloadRecordingWithRetry(ctx, downloadURLs, 2*time.Minute)
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
	report.Status = "ready"
	report.ProcessingState = "ready"
	report.RecordingStatus = "completed"
	report.TranscriptionStatus = "completed"
	report.AnalysisStatus = analysisStatus
	report.Duration = formatTranscriptTime(float64(max(30, len(lines)*30)))
	if report.Title == "" {
		report.Title = "Recording - " + roomName
	}
	readyDetail := s.reportDetailFromAnalysis(report, analysis)
	readyDetail.RecordingURL = reportStreamURL(reportID)
	readyDetail.RecordingSourceURL = firstNonEmpty(detail.RecordingSourceURL, downloadURL, playbackURL)
	readyDetail.RoomName = roomName
	s.storeReport(readyDetail)
	s.publishRoomRecordingReady(roomName, reportID)
}

func (s *Server) upsertEgressRecordingDetail(reportID, roomName, playbackURL, sourceURL string, now time.Time) reportDetailResponse {
	if existing, ok := s.reportDetailForUpdate(reportID); ok {
		existing.RecordingURL = firstNonEmpty(existing.RecordingURL, reportStreamURL(reportID))
		existing.RecordingSourceURL = firstNonEmpty(sourceURL, existing.RecordingSourceURL, playbackURL)
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
		RecordingURL:       reportStreamURL(reportID),
		RecordingSourceURL: firstNonEmpty(sourceURL, playbackURL),
		RoomName:           roomName,
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

func reportStreamURL(reportID string) string {
	reportID = strings.TrimSpace(reportID)
	if reportID == "" {
		return ""
	}
	return "/api/reports/" + reportID + "/recording/stream"
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
	urls := s.recordingDownloadURLs(state, info)
	if len(urls) == 0 {
		return ""
	}
	return urls[0]
}

func (s *Server) recordingDownloadURLs(state livekitservice.EgressState, info *lkproto.EgressInfo) []string {
	urls := make([]string, 0, 4)
	if url := s.s3ObjectURL(s.cfg.LiveKitS3Endpoint, s.cfg.LiveKitS3Bucket, firstEgressFilePath(state, info)); url != "" {
		urls = append(urls, url)
	}
	for _, value := range []string{state.FileLocation, state.PublicURL} {
		if isHTTPURL(value) {
			urls = append(urls, value)
		}
	}
	if info != nil {
		for _, file := range info.GetFileResults() {
			if isHTTPURL(file.GetLocation()) {
				urls = append(urls, file.GetLocation())
			}
		}
	}
	return uniqueStrings(urls)
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
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

var ErrRecordingDownloadFailed = errors.New("recording download failed")

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
		return "", "", nil, fmt.Errorf("%w: %d", ErrRecordingDownloadFailed, resp.StatusCode)
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

func downloadRecordingWithRetry(ctx context.Context, urls []string, maxWait time.Duration) (string, string, []byte, string, error) {
	startedAt := time.Now()
	var lastErr error
	for attempt := 1; ; attempt++ {
		for _, url := range urls {
			fileName, contentType, data, err := downloadRecording(ctx, url)
			if err == nil {
				if attempt > 1 {
					log.Printf("recording download succeeded after %d attempts using %s", attempt, url)
				}
				return fileName, contentType, data, url, nil
			}
			lastErr = err
			log.Printf("recording download not ready yet, retrying: attempt=%d url=%s err=%v", attempt, url, err)
		}
		if ctx.Err() != nil || (maxWait > 0 && time.Since(startedAt) >= maxWait) {
			return "", "", nil, "", lastErr
		}
		timer := time.NewTimer(time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return "", "", nil, "", ctx.Err()
		case <-timer.C:
		}
	}
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
