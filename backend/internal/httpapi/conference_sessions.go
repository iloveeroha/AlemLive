package httpapi

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type meetingSession struct {
	ReportID     string
	RoomName     string
	StartedAt    time.Time
	Participants map[string]struct{}
}

type conferenceEventResult struct {
	ReportID             string `json:"reportId,omitempty"`
	Participants         int    `json:"participants,omitempty"`
	RecordingShouldStop  bool   `json:"-"`
	ConferenceSaved      bool   `json:"conferenceSaved"`
	ConferenceStatus     string `json:"conferenceStatus,omitempty"`
	ConferenceReportPath string `json:"conferenceReportPath,omitempty"`
}

func (s *Server) recordConferenceEvent(roomName, userName, event string, now time.Time) conferenceEventResult {
	event = strings.ToLower(strings.TrimSpace(event))
	switch event {
	case "created", "joined", "left", "ended":
	default:
		return conferenceEventResult{}
	}

	roomName = strings.TrimSpace(firstNonEmpty(roomName, "alem-meeting"))
	userName = strings.TrimSpace(firstNonEmpty(userName, "Guest"))

	s.reportsMu.Lock()
	defer s.reportsMu.Unlock()

	session, ok := s.activeMeetings[roomName]
	if !ok {
		if resumed, resumedOK := s.resumeMeetingSessionLocked(roomName, userName, now); resumedOK {
			session = resumed
			ok = true
		}
	}
	if !ok {
		if event == "left" || event == "ended" {
			return conferenceEventResult{}
		}
		session = s.newMeetingSessionLocked(roomName, userName, now)
	}

	switch event {
	case "created", "joined":
		session.Participants[userName] = struct{}{}
		s.activeMeetings[roomName] = session
		s.updateConferenceReportLocked(session, now, "active")
	case "left":
		delete(session.Participants, userName)
		if len(session.Participants) == 0 {
			delete(s.activeMeetings, roomName)
			recordingShouldStop := s.conferenceRecordingShouldStopLocked(session.ReportID)
			finalStatus := s.finalConferenceStatusLocked(session.ReportID)
			s.updateConferenceReportLocked(session, now, finalStatus)
			return conferenceEventResult{
				ReportID:             session.ReportID,
				Participants:         0,
				RecordingShouldStop:  recordingShouldStop,
				ConferenceSaved:      true,
				ConferenceStatus:     finalStatus,
				ConferenceReportPath: "/api/reports/" + session.ReportID,
			}
		}
		s.activeMeetings[roomName] = session
		s.updateConferenceReportLocked(session, now, "active")
	case "ended":
		delete(s.activeMeetings, roomName)
		recordingShouldStop := s.conferenceRecordingShouldStopLocked(session.ReportID)
		finalStatus := s.finalConferenceStatusLocked(session.ReportID)
		s.updateConferenceReportLocked(session, now, finalStatus)
		return conferenceEventResult{
			ReportID:             session.ReportID,
			Participants:         len(session.Participants),
			RecordingShouldStop:  recordingShouldStop,
			ConferenceSaved:      true,
			ConferenceStatus:     finalStatus,
			ConferenceReportPath: "/api/reports/" + session.ReportID,
		}
	}

	return conferenceEventResult{
		ReportID:             session.ReportID,
		Participants:         len(session.Participants),
		ConferenceSaved:      true,
		ConferenceStatus:     "active",
		ConferenceReportPath: "/api/reports/" + session.ReportID,
	}
}

func (s *Server) finalConferenceStatusLocked(reportID string) string {
	detail, ok := s.generatedReportStore[reportID]
	if !ok {
		return "saved"
	}
	status := strings.ToLower(strings.TrimSpace(detail.Report.RecordingStatus))
	switch status {
	case "running", "completed":
		return "processing"
	case "failed":
		return "failed"
	default:
		return "saved"
	}
}

func (s *Server) conferenceRecordingShouldStopLocked(reportID string) bool {
	detail, ok := s.generatedReportStore[reportID]
	if !ok {
		return false
	}
	return strings.EqualFold(detail.Report.RecordingStatus, "running")
}

func (s *Server) newMeetingSessionLocked(roomName, userName string, now time.Time) meetingSession {
	if session, ok := s.resumeMeetingSessionLocked(roomName, userName, now); ok {
		return session
	}

	reportID := fmt.Sprintf("meeting-%s-%s", sanitizeReportID(roomName), now.UTC().Format("20060102-150405"))
	report := reportRow{
		ID:                  reportID,
		Title:               "Встреча - " + roomName,
		Source:              "LiveKit",
		Type:                "meeting",
		Date:                now.Format("02.01.2006"),
		Time:                now.Format("15:04"),
		Participants:        0,
		ParticipantNames:    "",
		Score:               0,
		Folder:              "Сохранённые встречи",
		Owner:               userName,
		OwnerInitial:        strings.ToUpper(firstNonEmpty(firstRune(userName), "A")),
		ThumbnailTone:       "blue",
		Week:                "LiveKit meetings",
		Duration:            "00:00",
		Status:              "active",
		ProcessingState:     "active",
		RecordingStatus:     "missing",
		TranscriptionStatus: "not_started",
		AnalysisStatus:      "not_started",
		CreatedAt:           now.UTC().Format(time.RFC3339),
		OccurredAt:          now,
	}
	detail := reportDetailResponse{
		Report:          report,
		Summary:         conferenceSummary(roomName, "recording", s.egress != nil && s.egress.Configured()),
		ActionItems:     []reportActionItem{},
		TranscriptLines: []reportTranscript{},
		Transcript:      []reportTranscript{},
		SpeakerStats:    []speakerStat{},
		Highlights:      []highlight{},
		Chapters:        []chapter{},
		AIQuestions: []string{
			"Какие решения приняли на встрече?",
			"Какие задачи появились после встречи?",
			"Сделай краткое резюме этой встречи.",
		},
		RoomName: roomName,
	}
	s.generatedReports = append([]reportRow{report}, s.generatedReports...)
	s.generatedReportStore[reportID] = detail
	s.latestRoomReports[roomName] = reportID

	return meetingSession{
		ReportID:     reportID,
		RoomName:     roomName,
		StartedAt:    now,
		Participants: map[string]struct{}{userName: struct{}{}},
	}
}

func (s *Server) resumeMeetingSessionLocked(roomName, userName string, now time.Time) (meetingSession, bool) {
	reportID := s.latestRoomReports[roomName]
	if reportID == "" {
		return meetingSession{}, false
	}
	detail, ok := s.generatedReportStore[reportID]
	if !ok {
		return meetingSession{}, false
	}
	state := strings.ToLower(strings.TrimSpace(detail.Report.ProcessingState))
	if state != "active" && state != "recording" {
		return meetingSession{}, false
	}

	participants := participantsFromReport(detail.Report.ParticipantNames)
	participants[userName] = struct{}{}
	startedAt := detail.Report.OccurredAt
	if startedAt.IsZero() {
		startedAt = now
	}
	return meetingSession{
		ReportID:     reportID,
		RoomName:     roomName,
		StartedAt:    startedAt,
		Participants: participants,
	}, true
}

func (s *Server) updateConferenceReportLocked(session meetingSession, now time.Time, status string) {
	detail, ok := s.generatedReportStore[session.ReportID]
	if !ok {
		return
	}

	participantNames := sortedParticipantNames(session.Participants)
	detail.Report.Participants = len(participantNames)
	detail.Report.ParticipantNames = strings.Join(participantNames, ", ")
	if session.StartedAt.IsZero() {
		session.StartedAt = now
	}
	if now.After(session.StartedAt) {
		detail.Report.Duration = formatTranscriptTime(now.Sub(session.StartedAt).Seconds())
	}
	detail.Report.Status = status
	detail.Report.ProcessingState = status
	switch status {
	case "active":
		if detail.Report.RecordingStatus == "" || detail.Report.RecordingStatus == "missing" {
			detail.Report.RecordingStatus = "missing"
		}
		if detail.Report.TranscriptionStatus == "" || detail.Report.TranscriptionStatus == "pending" {
			detail.Report.TranscriptionStatus = "not_started"
		}
		if detail.Report.AnalysisStatus == "" || detail.Report.AnalysisStatus == "pending" {
			detail.Report.AnalysisStatus = "not_started"
		}
	case "recording":
		detail.Report.RecordingStatus = "running"
		detail.Report.TranscriptionStatus = "pending"
		detail.Report.AnalysisStatus = "pending"
	case "processing":
		detail.Report.RecordingStatus = "completed"
		detail.Report.TranscriptionStatus = "pending"
		detail.Report.AnalysisStatus = "pending"
	case "saved":
		if detail.Report.RecordingStatus == "" || detail.Report.RecordingStatus == "running" {
			detail.Report.RecordingStatus = "missing"
		}
		detail.Report.TranscriptionStatus = "not_started"
		detail.Report.AnalysisStatus = "not_started"
	case "failed":
		if detail.Report.RecordingStatus == "" || detail.Report.RecordingStatus == "running" {
			detail.Report.RecordingStatus = "failed"
		}
		if detail.Report.TranscriptionStatus == "" || detail.Report.TranscriptionStatus == "pending" {
			detail.Report.TranscriptionStatus = "not_started"
		}
		if detail.Report.AnalysisStatus == "" || detail.Report.AnalysisStatus == "pending" {
			detail.Report.AnalysisStatus = "not_started"
		}
	}
	detail.Summary = conferenceSummary(session.RoomName, status, s.egress != nil && s.egress.Configured())
	detail.RoomName = session.RoomName
	detail.Transcript = detail.TranscriptLines

	s.generatedReportStore[session.ReportID] = detail
	for i, row := range s.generatedReports {
		if row.ID == session.ReportID {
			s.generatedReports[i] = detail.Report
			break
		}
	}
	s.latestRoomReports[session.RoomName] = session.ReportID
	s.saveReportsLocked()
}

func (s *Server) setConferenceReportPipelineState(reportID, roomName, state string) {
	reportID = strings.TrimSpace(reportID)
	roomName = strings.TrimSpace(roomName)
	if reportID == "" && roomName != "" {
		reportID = s.latestReportIDForRoom(roomName)
	}
	if reportID == "" {
		return
	}

	s.reportsMu.Lock()
	defer s.reportsMu.Unlock()

	detail, ok := s.generatedReportStore[reportID]
	if !ok {
		return
	}
	roomName = firstNonEmpty(roomName, detail.RoomName)
	detail.RoomName = roomName
	switch state {
	case roomRecordingRecording:
		detail.Report.Status = "recording"
		detail.Report.ProcessingState = "recording"
		detail.Report.RecordingStatus = "running"
		detail.Report.TranscriptionStatus = "pending"
		detail.Report.AnalysisStatus = "pending"
		detail.Summary = conferenceSummary(roomName, "recording", s.egress != nil && s.egress.Configured())
	case roomRecordingProcessing:
		detail.Report.Status = "processing"
		detail.Report.ProcessingState = "processing"
		detail.Report.RecordingStatus = "completed"
		detail.Report.TranscriptionStatus = "pending"
		detail.Report.AnalysisStatus = "pending"
		detail.Summary = conferenceSummary(roomName, "processing", s.egress != nil && s.egress.Configured())
	case roomRecordingError:
		detail.Report.Status = "failed"
		detail.Report.ProcessingState = "failed"
		detail.Report.RecordingStatus = "failed"
		if detail.Report.TranscriptionStatus == "" || detail.Report.TranscriptionStatus == "pending" {
			detail.Report.TranscriptionStatus = "not_started"
		}
		if detail.Report.AnalysisStatus == "" || detail.Report.AnalysisStatus == "pending" {
			detail.Report.AnalysisStatus = "not_started"
		}
		detail.Summary = conferenceSummary(roomName, "failed", s.egress != nil && s.egress.Configured())
	default:
		return
	}

	s.generatedReportStore[reportID] = detail
	for i, row := range s.generatedReports {
		if row.ID == reportID {
			s.generatedReports[i] = detail.Report
			break
		}
	}
	if roomName != "" {
		s.latestRoomReports[roomName] = reportID
	}
	s.saveReportsLocked()
}

func (s *Server) latestReportIDForRoom(roomName string) string {
	s.reportsMu.Lock()
	defer s.reportsMu.Unlock()
	return s.latestRoomReports[roomName]
}

func (s *Server) reportDetailForUpdate(reportID string) (reportDetailResponse, bool) {
	s.reportsMu.Lock()
	defer s.reportsMu.Unlock()
	detail, ok := s.generatedReportStore[reportID]
	return detail, ok
}

func sortedParticipantNames(participants map[string]struct{}) []string {
	names := make([]string, 0, len(participants))
	for name := range participants {
		if strings.TrimSpace(name) != "" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func participantsFromReport(value string) map[string]struct{} {
	participants := map[string]struct{}{}
	for _, part := range strings.Split(value, ",") {
		name := strings.TrimSpace(part)
		if name != "" {
			participants[name] = struct{}{}
		}
	}
	return participants
}

func legacyConferenceSummary(roomName, status string, egressEnabled bool) []summarySection {
	if status == "saved" {
		text := "Конференция сохранена в отчётах."
		if egressEnabled {
			text += " Запись, транскрипт и AI-анализ будут добавлены после завершения LiveKit Egress."
		} else {
			text += " Для видео и автоматического STT включите LiveKit Egress и storage."
		}
		return []summarySection{{Title: "Конференция сохранена", Text: text}}
	}

	text := "Backend уже создал отчёт для комнаты " + roomName + " и обновляет его по событиям участников."
	if egressEnabled {
		text += " LiveKit Egress записывает конференцию автоматически."
	}
	return []summarySection{{Title: "Конференция записывается", Text: text}}
}
