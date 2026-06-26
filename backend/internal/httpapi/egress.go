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
	"sort"
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
		response["active"] = state.EgressID != "" && isLiveKitRecordingActiveStatus(state.Status)
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
	mode := normalizedRecordingMode(s.cfg.RecordingMode)
	if mode == livekitservice.RecordingModeParticipantTracks {
		tracks := s.roomAudioTracks(roomName)
		if len(tracks) > 0 {
			state, err := s.egress.StartParticipantTracks(ctx, roomName, tracks, s.clock())
			if err == nil {
				return state, nil
			}
			if strings.TrimSpace(s.cfg.RecordingFallbackMode) == "" || strings.EqualFold(s.cfg.RecordingFallbackMode, livekitservice.RecordingModeRoomComposite) {
				fallback, fallbackErr := s.egress.StartRoomComposite(ctx, roomName, s.clock())
				if fallbackErr != nil {
					return state, err
				}
				fallback.RecordingMode = livekitservice.RecordingModeParticipantFallback
				fallback.FallbackReason = "participant track egress failed; room composite recording was started"
				return fallback, nil
			}
			return state, err
		}
		state, err := s.egress.StartRoomComposite(ctx, roomName, s.clock())
		if err != nil {
			return state, err
		}
		state.RecordingMode = livekitservice.RecordingModeParticipantFallback
		state.FallbackReason = "participant audio tracks are not available yet; room composite recording was started"
		return state, nil
	}
	state, err := s.egress.StartRoomComposite(ctx, roomName, s.clock())
	if state.RecordingMode == "" {
		state.RecordingMode = livekitservice.RecordingModeRoomComposite
	}
	return state, err
}

func normalizedRecordingMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case livekitservice.RecordingModeParticipantTracks:
		return livekitservice.RecordingModeParticipantTracks
	case livekitservice.RecordingModeRoomComposite, "":
		return livekitservice.RecordingModeRoomComposite
	default:
		return livekitservice.RecordingModeRoomComposite
	}
}

func (s *Server) stopRoomRecording(ctx context.Context, roomName string) (livekitservice.EgressState, error) {
	if s.egress == nil || !s.egress.Configured() {
		return livekitservice.EgressState{}, livekitservice.ErrEgressNotConfigured
	}
	// StopEgress must finish even if the browser closes the HTTP request while
	// LiveKit is disconnecting the page. Otherwise the UI may show "stop" as
	// clicked, but the recording keeps running or later lands in an error state.
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
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
				s.recordEgressPipelineError(reportID, "recording", "LiveKit Egress start failed: "+recordingErrorMessage(err))
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
		s.recordEgressPipelineState(reportID, state, "recording", "running", "LiveKit Egress started")
		if state.FallbackReason != "" {
			s.addReportWarning(reportID, state.FallbackReason)
		}
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
				s.recordEgressPipelineError(reportID, "recording", "LiveKit Egress stop failed: "+recordingErrorMessage(err))
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
		s.recordEgressPipelineState(reportID, state, "recording", "stopping", "LiveKit Egress stopped; waiting for recording file")
		if state.FallbackReason != "" {
			s.addReportWarning(reportID, state.FallbackReason)
		}
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
	case strings.Contains(status, "stop") || strings.Contains(status, "process") || strings.Contains(status, "finaliz"):
		return roomRecordingProcessing
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

func isLiveKitRecordingActiveStatus(status string) bool {
	status = strings.ToLower(strings.TrimSpace(status))
	if status == "" {
		return true
	}
	return !strings.Contains(status, "stop") &&
		!strings.Contains(status, "process") &&
		!strings.Contains(status, "finaliz") &&
		!strings.Contains(status, "complete") &&
		!strings.Contains(status, "fail") &&
		!strings.Contains(status, "abort") &&
		!strings.Contains(status, "limit")
}

func recordingErrorMessage(err error) string {
	if errors.Is(err, livekitservice.ErrEgressNotConfigured) {
		return "LiveKit Egress is not configured"
	}
	message := strings.TrimSpace(err.Error())
	if message == "" {
		return "LiveKit Egress request failed"
	}
	message = strings.NewReplacer("\r", " ", "\n", " ", "\t", " ").Replace(message)
	return "LiveKit Egress request failed: " + truncateRunes(message, 240)
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
	if participant := event.GetParticipant(); participant != nil {
		s.handleLiveKitParticipantWebhook(event.GetEvent(), event.GetRoom(), participant, event.GetTrack())
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

func (s *Server) handleLiveKitParticipantWebhook(eventName string, room *lkproto.Room, participant *lkproto.ParticipantInfo, track *lkproto.TrackInfo) {
	if participant == nil {
		return
	}
	roomName := strings.TrimSpace(room.GetName())
	if roomName == "" {
		return
	}
	user := roomUser{
		ID:   strings.TrimSpace(participant.GetIdentity()),
		Name: strings.TrimSpace(participant.GetName()),
	}
	if metadata := liveKitParticipantMetadata(participant.GetMetadata()); len(metadata) > 0 {
		user.Name = firstNonEmpty(metadata["display_name"], metadata["name"], user.Name, user.ID)
		user.Email = firstNonEmpty(metadata["email"], user.Email)
	}
	if user.ID == "" {
		user.ID = firstNonEmpty(user.Name, participant.GetSid())
	}
	if user.Name == "" {
		user.Name = user.ID
	}

	audioTrackID := ""
	if track != nil && isLiveKitAudioTrack(track) {
		audioTrackID = firstNonEmpty(track.GetSid(), track.GetName())
	}
	snapshot := s.upsertRoomParticipantFromLiveKit(roomName, user, eventName, audioTrackID, s.clock().UTC())
	reportID := firstNonEmpty(snapshot.ReportID, s.latestReportIDForRoom(roomName))
	if reportID != "" {
		s.syncReportParticipantsFromSnapshot(reportID, snapshot)
	}
	if strings.EqualFold(eventName, "track_published") && audioTrackID != "" {
		s.maybeStartParticipantTrackRecording(roomName, user, audioTrackID, reportID)
	}
}

func liveKitParticipantMetadata(raw string) map[string]string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	values := map[string]string{}
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil
	}
	return values
}

func isLiveKitAudioTrack(track *lkproto.TrackInfo) bool {
	if track == nil {
		return false
	}
	if track.GetType() == lkproto.TrackType_AUDIO {
		return true
	}
	return track.GetSource() == lkproto.TrackSource_MICROPHONE
}

func (s *Server) maybeStartParticipantTrackRecording(roomName string, user roomUser, trackID, reportID string) {
	if s.egress == nil || !s.egress.Configured() {
		return
	}
	state, ok := s.egress.Status(roomName)
	if !ok || state.RecordingMode != livekitservice.RecordingModeParticipantTracks || !isLiveKitRecordingActiveStatus(state.Status) {
		return
	}
	for _, existing := range state.TrackEgresses {
		if strings.TrimSpace(existing.TrackID) == strings.TrimSpace(trackID) {
			return
		}
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		updated, err := s.egress.AddParticipantTrack(ctx, roomName, livekitservice.ParticipantTrack{
			ParticipantID:       user.ID,
			ParticipantIdentity: user.ID,
			ParticipantName:     user.Name,
			TrackID:             trackID,
		}, s.clock())
		if err != nil {
			log.Printf("participant track egress skipped: room=%s track=%s participant=%s error=%v", roomName, trackID, user.ID, err)
			if reportID != "" {
				s.addReportWarning(reportID, "participant audio track could not be recorded for "+firstNonEmpty(user.Name, user.ID, "participant"))
			}
			return
		}
		if reportID != "" {
			s.recordEgressPipelineState(reportID, updated, "speaker_tracks", "running", "Participant audio track recording started for "+firstNonEmpty(user.Name, user.ID, "participant"))
		}
		roomID := roomIDFromName(roomName)
		s.broadcastRoomEvent(roomID, roomEventEnvelope{
			Type: "recording_status_changed",
			Payload: map[string]any{
				"roomId":    roomID,
				"roomName":  roomName,
				"state":     roomRecordingRecording,
				"status":    roomRecordingRecording,
				"recording": updated,
				"reportId":  reportID,
			},
		})
	}()
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
	log.Printf("egress report %s processing started: room=%s egress=%s status=%s filePath=%s", reportID, roomName, state.EgressID, state.Status, state.FilePath)
	downloadURL := s.firstRecordingDownloadURL(state, info)
	if downloadURL == "" {
		log.Printf("egress report %s skipped: no recording download URL for room=%s egress=%s", reportID, state.RoomName, state.EgressID)
		s.setConferenceReportPipelineState(reportID, roomName, roomRecordingError)
		s.recordEgressPipelineError(reportID, "recording", "Egress completed but backend could not determine recording object URL")
		return
	}
	detail := s.upsertEgressRecordingDetail(reportID, roomName, playbackURL, downloadURL, now)
	s.applyEgressRecordingArtifacts(&detail, state, info)

	if s.stt == nil || !s.stt.Configured() {
		log.Printf("egress report %s saved without STT: STT is not configured", reportID)
		detail.Report.Status = "ready"
		detail.Report.ProcessingState = "ready"
		detail.Report.RecordingStatus = "completed"
		detail.Report.TranscriptionStatus = "not_configured"
		detail.Report.AnalysisStatus = "not_started"
		detail.SpeakerSource = speakerSourceUnknown
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
		s.recordEgressPipelineError(reportID, "recording", "Egress completed but no MinIO/S3 recording URL was available")
		return
	}

	s.recordEgressPipelineState(reportID, state, "recording", "verifying", "Verifying recording file before STT")
	log.Printf("egress report %s verifying recording file before STT", reportID)
	fileName, contentType, data, downloadURL, err := downloadRecordingWithRetry(ctx, downloadURLs, 2*time.Minute)
	if err != nil {
		log.Printf("egress report %s recording download failed: %v", reportID, err)
		s.markEgressReportFailed(reportID, err)
		return
	}
	log.Printf("egress report %s recording file verified: file=%s contentType=%s bytes=%d", reportID, fileName, contentType, len(data))
	s.recordEgressPipelineState(reportID, state, "recording", "completed", fmt.Sprintf("Recording file verified: %s (%d bytes)", fileName, len(data)))
	s.recordEgressPipelineState(reportID, state, "transcription", "running", "Starting speech-to-text")
	participants := strings.TrimSpace(detail.Report.ParticipantNames)
	transcriptText, lines, speakerSource, err := s.transcribeEgressRecording(ctx, reportID, state, fileName, contentType, data)
	if err != nil {
		log.Printf("egress report %s transcription failed: %v", reportID, err)
		s.markEgressReportFailed(reportID, err)
		return
	}
	if speakerSource == speakerSourceParticipantTrack {
		s.recordEgressPipelineState(reportID, state, "diarization", "not_needed", "Participant audio tracks provided speaker attribution")
	} else {
		if s.cfg.EnableDiarizationFallback && s.diarizer != nil && s.diarizer.Configured() {
			s.recordEgressPipelineState(reportID, state, "diarization", "running", "Starting speaker diarization")
		} else {
			s.recordEgressPipelineState(reportID, state, "diarization", "not_configured", "Diarization service is not configured")
		}
		lines = s.diarizeTranscript(ctx, fileName, contentType, data, participants, lines)
		if s.cfg.EnableDiarizationFallback && s.diarizer != nil && s.diarizer.Configured() {
			s.recordEgressPipelineState(reportID, state, "diarization", "completed", "Speaker diarization completed")
		}
	}
	s.recordEgressPipelineState(reportID, state, "analysis", "running", "Starting LLM meeting analysis")
	analysis, err := s.generateMeetingAnalysisFromTranscript(ctx, roomName, participants, transcriptText, lines)
	analysisStatus := "completed"
	if err != nil {
		log.Printf("egress report %s meeting analysis fallback: %v", reportID, err)
		analysis = fallbackAnalysisFromTranscript(roomName, transcriptText, lines, s.clock())
		analysisStatus = "failed"
		s.recordEgressPipelineError(reportID, "analysis", "LLM analysis failed: "+err.Error())
	} else {
		s.recordEgressPipelineState(reportID, state, "analysis", "completed", "LLM meeting analysis completed")
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
	copyReportArtifacts(&readyDetail, detail)
	s.applyEgressRecordingArtifacts(&readyDetail, state, info)
	readyDetail.SpeakerSource = firstNonEmpty(speakerSource, defaultSpeakerSource(analysis.Transcript))
	readyDetail.RecordingURL = reportStreamURL(reportID)
	readyDetail.RecordingSourceURL = firstNonEmpty(detail.RecordingSourceURL, downloadURL, playbackURL)
	readyDetail.EgressID = firstNonEmpty(detail.EgressID, state.EgressID)
	readyDetail.RecordingBucket = firstNonEmpty(detail.RecordingBucket, s.cfg.LiveKitS3Bucket)
	readyDetail.RecordingObjectKey = firstNonEmpty(detail.RecordingObjectKey, firstEgressFilePath(state, info))
	readyDetail.RecordingStartedAt = firstNonEmpty(detail.RecordingStartedAt, state.StartedAt)
	readyDetail.RecordingCompletedAt = firstNonEmpty(detail.RecordingCompletedAt, state.EndedAt, s.clock().UTC().Format(time.RFC3339))
	if s.cfg.EnableDiarizationFallback && s.diarizer != nil && s.diarizer.Configured() {
		readyDetail.DiarizationStatus = firstNonEmpty(detail.DiarizationStatus, "completed")
	} else {
		readyDetail.DiarizationStatus = firstNonEmpty(detail.DiarizationStatus, "not_configured")
	}
	readyDetail.RoomName = roomName
	s.storeReport(readyDetail)
	log.Printf("egress report %s processing finished: analysis=%s transcriptLines=%d", reportID, analysisStatus, len(lines))
	s.publishRoomRecordingReady(roomName, reportID)
}

func (s *Server) transcribeEgressRecording(ctx context.Context, reportID string, state livekitservice.EgressState, fileName, contentType string, data []byte) (string, []transcriptLine, string, error) {
	if len(state.TrackEgresses) > 0 {
		s.recordEgressPipelineState(reportID, state, "speaker_tracks", "running", "Starting speech-to-text for participant audio tracks")
		transcriptText, lines, err := s.transcribeParticipantTrackRecordings(ctx, reportID, state)
		if err == nil && len(lines) > 0 {
			s.recordEgressPipelineState(reportID, state, "speaker_tracks", "completed", fmt.Sprintf("Participant audio tracks transcribed: %d lines", len(lines)))
			s.recordEgressPipelineState(reportID, state, "transcription", "completed", "Speech-to-text completed from participant audio tracks")
			return transcriptText, lines, speakerSourceParticipantTrack, nil
		}
		if err != nil {
			log.Printf("egress report %s participant track transcription fallback: %v", reportID, err)
			s.recordEgressPipelineState(reportID, state, "speaker_tracks", "failed", "Participant audio track transcription failed; falling back to room composite audio")
		}
	}

	log.Printf("egress report %s starting speech-to-text using model=%s", reportID, s.cfg.STTModel)
	transcription, err := s.stt.Transcribe(ctx, fileName, contentType, data, llm.TranscriptionOptions{
		Model:          s.cfg.STTModel,
		ResponseFormat: "verbose_json",
	})
	if err != nil {
		return "", nil, "", err
	}
	s.recordEgressPipelineState(reportID, state, "transcription", "completed", "Speech-to-text completed")
	transcriptText := truncateRunes(strings.TrimSpace(transcription.Text), maxTranscriptRunes)
	lines := transcriptLinesFromTranscription(transcription)
	if len(lines) == 0 {
		lines = transcriptLinesFromText(transcriptText)
	}
	return transcriptText, lines, speakerSourceSTT, nil
}

func (s *Server) transcribeParticipantTrackRecordings(ctx context.Context, reportID string, state livekitservice.EgressState) (string, []transcriptLine, error) {
	var allLines []transcriptLine
	var transcriptParts []string
	var lastErr error

	for _, track := range state.TrackEgresses {
		if track.TrackID == "" {
			continue
		}
		urls := s.trackRecordingDownloadURLs(track)
		if len(urls) == 0 {
			lastErr = fmt.Errorf("no recording URL for participant audio track %s", track.TrackID)
			continue
		}
		fileName, contentType, data, _, err := downloadRecordingWithRetry(ctx, urls, 2*time.Minute)
		if err != nil {
			lastErr = err
			log.Printf("egress report %s participant track %s download failed: %v", reportID, track.TrackID, err)
			continue
		}
		s.recordEgressPipelineState(reportID, state, "speaker_tracks", "verified", fmt.Sprintf("Participant audio track verified: %s (%d bytes)", fileName, len(data)))
		transcription, err := s.stt.Transcribe(ctx, fileName, contentType, data, llm.TranscriptionOptions{
			Model:          s.cfg.STTModel,
			ResponseFormat: "verbose_json",
		})
		if err != nil {
			lastErr = err
			log.Printf("egress report %s participant track %s STT failed: %v", reportID, track.TrackID, err)
			continue
		}
		lines := transcriptLinesFromTranscription(transcription)
		if len(lines) == 0 {
			lines = transcriptLinesFromText(transcription.Text)
		}
		lines = assignTranscriptLinesToTrack(lines, track, len(allLines))
		allLines = append(allLines, lines...)
		if text := strings.TrimSpace(transcription.Text); text != "" {
			transcriptParts = append(transcriptParts, firstNonEmpty(track.ParticipantName, track.ParticipantIdentity, track.ParticipantID, "Speaker")+": "+text)
		}
	}

	if len(allLines) == 0 {
		if lastErr == nil {
			lastErr = errors.New("participant audio tracks were not available for transcription")
		}
		return "", nil, lastErr
	}

	sort.SliceStable(allLines, func(i, j int) bool {
		if allLines[i].Start == allLines[j].Start {
			return strings.TrimSpace(allLines[i].Speaker) < strings.TrimSpace(allLines[j].Speaker)
		}
		return allLines[i].Start < allLines[j].Start
	})
	for i := range allLines {
		if allLines[i].ID == "" {
			allLines[i].ID = fmt.Sprintf("seg-%d", i+1)
		}
	}
	return truncateRunes(strings.Join(transcriptParts, "\n"), maxTranscriptRunes), allLines, nil
}

func assignTranscriptLinesToTrack(lines []transcriptLine, track livekitservice.TrackEgressState, offset int) []transcriptLine {
	speaker := normalizeSpeakerLabel(firstNonEmpty(track.ParticipantName, track.ParticipantIdentity, track.ParticipantID, "Speaker"))
	out := make([]transcriptLine, len(lines))
	for i, line := range lines {
		line.ID = fmt.Sprintf("%s-seg-%d", sanitizeReportID(firstNonEmpty(track.TrackID, "track")), offset+i+1)
		line.Speaker = speaker
		line.SpeakerName = speaker
		line.SpeakerID = speakerIDFromName(speaker)
		line.ParticipantID = firstNonEmpty(track.ParticipantID, track.ParticipantIdentity)
		line.LiveKitIdentity = firstNonEmpty(track.ParticipantIdentity, track.ParticipantID)
		line.TrackID = track.TrackID
		line.Source = speakerSourceParticipantTrack
		if line.Time == "" {
			line.Time = formatTranscriptTime(line.Start)
		}
		out[i] = line
	}
	return out
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
		existing.RecordingBucket = firstNonEmpty(existing.RecordingBucket, s.cfg.LiveKitS3Bucket)
		if existing.LastError != "" {
			existing.LastError = ""
		}
		addPipelineEvent(&existing, "recording", "completed", "LiveKit Egress completed and recording URL was found", now)
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
		RecordingBucket:    s.cfg.LiveKitS3Bucket,
		RoomName:           roomName,
	}
	addPipelineEvent(&detail, "recording", "completed", "LiveKit Egress completed and recording URL was found", now)
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
	message := reportProcessingErrorMessage(err)
	if errors.Is(err, ErrRecordingDownloadFailed) {
		detail.Report.RecordingStatus = "failed"
		detail.Report.TranscriptionStatus = "not_started"
		detail.Report.AnalysisStatus = "not_started"
		detail.RecordingError = message
		detail.LastError = message
		detail.Summary = []summarySection{
			{Title: "Recording failed", Text: "LiveKit Egress completed, but backend could not verify/download the recording file from MinIO. STT, diarization and AI analysis were not started. " + message},
		}
		addPipelineEvent(&detail, "recording", "failed", message, s.clock())
		s.storeReport(detail)
		s.publishRoomRecordingReady(detail.RoomName, reportID)
		return
	}

	if detail.Report.RecordingStatus == "" {
		detail.Report.RecordingStatus = "completed"
	}
	detail.Report.TranscriptionStatus = "failed"
	if detail.Report.AnalysisStatus == "" {
		detail.Report.AnalysisStatus = "not_started"
	}
	detail.TranscriptionError = message
	detail.LastError = message
	detail.Summary = []summarySection{
		{Title: "Recording captured", Text: "Meeting video is available, but transcript/AI processing failed: " + message},
	}
	addPipelineEvent(&detail, "transcription", "failed", message, s.clock())
	s.storeReport(detail)
	s.publishRoomRecordingReady(detail.RoomName, reportID)
}

func (s *Server) recordEgressPipelineState(reportID string, state livekitservice.EgressState, stage, status, message string) {
	if strings.TrimSpace(reportID) == "" {
		return
	}
	detail, ok := s.reportDetailForUpdate(reportID)
	if !ok {
		return
	}
	if state.EgressID != "" {
		detail.EgressID = state.EgressID
	}
	if state.StartedAt != "" {
		detail.RecordingStartedAt = state.StartedAt
	}
	if state.EndedAt != "" {
		detail.RecordingCompletedAt = state.EndedAt
	}
	if state.FilePath != "" && !isHTTPURL(state.FilePath) {
		detail.RecordingObjectKey = state.FilePath
	}
	detail.RecordingBucket = firstNonEmpty(detail.RecordingBucket, s.cfg.LiveKitS3Bucket)
	addPipelineEvent(&detail, stage, status, message, s.clock())
	s.storeReport(detail)
}

func (s *Server) recordEgressPipelineError(reportID, stage, message string) {
	if strings.TrimSpace(reportID) == "" {
		return
	}
	detail, ok := s.reportDetailForUpdate(reportID)
	if !ok {
		return
	}
	message = strings.TrimSpace(message)
	detail.LastError = message
	switch stage {
	case "recording":
		detail.RecordingError = message
	case "diarization":
		detail.DiarizationError = message
	case "analysis":
		detail.AnalysisError = message
	default:
		detail.TranscriptionError = message
	}
	addPipelineEvent(&detail, stage, "failed", message, s.clock())
	s.storeReport(detail)
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

func (s *Server) applyEgressRecordingArtifacts(detail *reportDetailResponse, state livekitservice.EgressState, info *lkproto.EgressInfo) {
	if detail == nil {
		return
	}
	mode := firstNonEmpty(state.RecordingMode, detail.RecordingMode, normalizedRecordingMode(s.cfg.RecordingMode))
	detail.RecordingMode = mode
	if detail.SpeakerSource == "" {
		switch mode {
		case livekitservice.RecordingModeParticipantTracks:
			detail.SpeakerSource = speakerSourceParticipantTrack
		case livekitservice.RecordingModeParticipantFallback:
			detail.SpeakerSource = speakerSourceDiarization
		default:
			detail.SpeakerSource = speakerSourceDiarization
		}
	}
	if state.FallbackReason != "" {
		exists := false
		for _, warning := range detail.Warnings {
			if warning == state.FallbackReason {
				exists = true
				break
			}
		}
		if !exists {
			detail.Warnings = append(detail.Warnings, state.FallbackReason)
		}
	}
	objectKey := firstEgressFilePath(state, info)
	if objectKey != "" {
		file := recordingFile{
			RecordingMode: mode,
			Bucket:        firstNonEmpty(detail.RecordingBucket, s.cfg.LiveKitS3Bucket),
			ObjectKey:     objectKey,
			PublicURL:     firstNonEmpty(state.PublicURL, detail.RecordingSourceURL),
			MediaType:     recordingContentType(objectKey),
			Status:        firstNonEmpty(state.Status, detail.Report.RecordingStatus),
		}
		detail.RecordingFiles = mergeRecordingFiles(detail.RecordingFiles, file)
	}
	for _, track := range state.TrackEgresses {
		objectKey := track.FilePath
		if objectKey == "" || isHTTPURL(objectKey) {
			continue
		}
		detail.RecordingFiles = mergeRecordingFiles(detail.RecordingFiles, recordingFile{
			RecordingMode:       livekitservice.RecordingModeParticipantTracks,
			Bucket:              firstNonEmpty(detail.RecordingBucket, s.cfg.LiveKitS3Bucket),
			ObjectKey:           objectKey,
			PublicURL:           firstNonEmpty(track.PublicURL, track.FileLocation),
			ParticipantIdentity: firstNonEmpty(track.ParticipantIdentity, track.ParticipantID),
			TrackID:             track.TrackID,
			MediaType:           recordingContentType(objectKey),
			Status:              firstNonEmpty(track.Status, state.Status, detail.Report.RecordingStatus),
		})
	}
}

func mergeRecordingFiles(existing []recordingFile, incoming recordingFile) []recordingFile {
	key := strings.TrimSpace(incoming.ObjectKey)
	if key == "" {
		return existing
	}
	out := append([]recordingFile(nil), existing...)
	for i := range out {
		if strings.TrimSpace(out[i].ObjectKey) == key {
			out[i] = incoming
			return out
		}
	}
	return append(out, incoming)
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

func (s *Server) trackRecordingDownloadURLs(track livekitservice.TrackEgressState) []string {
	urls := make([]string, 0, 3)
	if url := s.s3ObjectURL(s.cfg.LiveKitS3Endpoint, s.cfg.LiveKitS3Bucket, track.FilePath); url != "" {
		urls = append(urls, url)
	}
	for _, value := range []string{track.FileLocation, track.PublicURL} {
		if isHTTPURL(value) {
			urls = append(urls, value)
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
		return "", "", nil, fmt.Errorf("%w: %v", ErrRecordingDownloadFailed, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", nil, fmt.Errorf("%w: %v", ErrRecordingDownloadFailed, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", nil, fmt.Errorf("%w: %d", ErrRecordingDownloadFailed, resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxRecordingUploadBytes+1))
	if err != nil {
		return "", "", nil, fmt.Errorf("%w: %v", ErrRecordingDownloadFailed, err)
	}
	if int64(len(data)) > maxRecordingUploadBytes {
		return "", "", nil, fmt.Errorf("%w: recording file is too large", ErrRecordingDownloadFailed)
	}
	if len(data) == 0 {
		return "", "", nil, fmt.Errorf("%w: empty recording file", ErrRecordingDownloadFailed)
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
			if lastErr != nil {
				return "", "", nil, "", lastErr
			}
			if ctx.Err() != nil {
				return "", "", nil, "", ctx.Err()
			}
			return "", "", nil, "", ErrRecordingDownloadFailed
		}
		timer := time.NewTimer(time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			if lastErr != nil {
				return "", "", nil, "", lastErr
			}
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
