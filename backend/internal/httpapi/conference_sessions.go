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
		session = s.newMeetingSessionLocked(roomName, userName, now)
		ok = true
	}

	switch event {
	case "created", "joined":
		session.Participants[userName] = struct{}{}
		s.activeMeetings[roomName] = session
		s.updateConferenceReportLocked(session, now, "recording")
	case "left":
		delete(session.Participants, userName)
		if len(session.Participants) == 0 {
			delete(s.activeMeetings, roomName)
			s.updateConferenceReportLocked(session, now, "saved")
			return conferenceEventResult{
				ReportID:             session.ReportID,
				Participants:         0,
				RecordingShouldStop:  true,
				ConferenceSaved:      true,
				ConferenceStatus:     "saved",
				ConferenceReportPath: "/api/reports/" + session.ReportID,
			}
		}
		s.activeMeetings[roomName] = session
		s.updateConferenceReportLocked(session, now, "recording")
	case "ended":
		delete(s.activeMeetings, roomName)
		s.updateConferenceReportLocked(session, now, "saved")
		return conferenceEventResult{
			ReportID:             session.ReportID,
			Participants:         len(session.Participants),
			RecordingShouldStop:  true,
			ConferenceSaved:      true,
			ConferenceStatus:     "saved",
			ConferenceReportPath: "/api/reports/" + session.ReportID,
		}
	}

	return conferenceEventResult{
		ReportID:             session.ReportID,
		Participants:         len(session.Participants),
		ConferenceSaved:      true,
		ConferenceStatus:     "recording",
		ConferenceReportPath: "/api/reports/" + session.ReportID,
	}
}

func (s *Server) newMeetingSessionLocked(roomName, userName string, now time.Time) meetingSession {
	if reportID := s.latestRoomReports[roomName]; reportID != "" {
		if detail, ok := s.generatedReportStore[reportID]; ok && detail.Report.ProcessingState == "recording" {
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
			}
		}
	}

	reportID := fmt.Sprintf("meeting-%s-%s", sanitizeReportID(roomName), now.UTC().Format("20060102-150405"))
	report := reportRow{
		ID:               reportID,
		Title:            "Встреча - " + roomName,
		Source:           "LiveKit",
		Type:             "meeting",
		Date:             now.Format("02.01.2006"),
		Time:             now.Format("15:04"),
		Participants:     0,
		ParticipantNames: "",
		Score:            0,
		Folder:           "Сохранённые встречи",
		Owner:            userName,
		OwnerInitial:     strings.ToUpper(firstNonEmpty(firstRune(userName), "A")),
		ThumbnailTone:    "blue",
		Week:             "LiveKit meetings",
		Duration:         "00:00",
		Status:           "recording",
		ProcessingState:  "recording",
		CreatedAt:        now.UTC().Format(time.RFC3339),
		OccurredAt:       now,
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

func conferenceSummary(roomName, status string, egressEnabled bool) []summarySection {
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
