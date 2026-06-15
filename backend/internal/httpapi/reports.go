package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/iloveeroha/AlemLive/backend/internal/llm"
)

type reportRow struct {
	ID               string    `json:"id"`
	Title            string    `json:"title"`
	Source           string    `json:"source"`
	Type             string    `json:"type"`
	Date             string    `json:"date"`
	Time             string    `json:"time"`
	Participants     int       `json:"participants"`
	ParticipantNames string    `json:"participantNames"`
	Score            int       `json:"score"`
	Folder           string    `json:"folder"`
	Owner            string    `json:"owner"`
	OwnerInitial     string    `json:"ownerInitial"`
	ThumbnailTone    string    `json:"thumbnailTone"`
	Week             string    `json:"week"`
	Duration         string    `json:"duration"`
	Status           string    `json:"status"`
	ProcessingState  string    `json:"processingState"`
	CreatedAt        string    `json:"createdAt"`
	OccurredAt       time.Time `json:"-"`
}

type reportsResponse struct {
	Reports []reportRow   `json:"reports"`
	Items   []reportRow   `json:"items"`
	Total   int           `json:"total"`
	Limit   int           `json:"limit"`
	Offset  int           `json:"offset"`
	Filters reportFilters `json:"filters"`
}

type reportFilters struct {
	Sources          []string          `json:"sources"`
	Folders          []string          `json:"folders"`
	Owners           []string          `json:"owners"`
	QuickDateOptions []quickDateOption `json:"quickDateOptions"`
}

type quickDateOption struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type reportDetailResponse struct {
	Report          reportRow          `json:"report"`
	Summary         []summarySection   `json:"summary"`
	ActionItems     []reportActionItem `json:"actionItems"`
	TranscriptLines []reportTranscript `json:"transcriptLines"`
	Transcript      []reportTranscript `json:"transcript"`
	SpeakerStats    []speakerStat      `json:"speakerStats"`
	Highlights      []highlight        `json:"highlights"`
	Chapters        []chapter          `json:"chapters"`
	AIQuestions     []string           `json:"aiQuestions"`
	RecordingURL    string             `json:"recordingUrl,omitempty"`
	RecordingFile   string             `json:"recordingFile,omitempty"`
	RecordingType   string             `json:"recordingType,omitempty"`
	RoomName        string             `json:"roomName,omitempty"`
}

type reportActionItem struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Task        string `json:"task"`
	Description string `json:"description"`
	Owner       string `json:"owner"`
	Due         string `json:"due"`
	Priority    string `json:"priority"`
	Status      string `json:"status"`
}

type reportTranscript struct {
	ID      string `json:"id"`
	Time    string `json:"time"`
	Speaker string `json:"speaker"`
	Text    string `json:"text"`
}

type speakerStat struct {
	Name         string `json:"name"`
	Role         string `json:"role"`
	TalkTime     int    `json:"talkTime"`
	TalkTimeText string `json:"talkTimeText"`
	Talk         int    `json:"talk"`
	Sentiment    string `json:"sentiment"`
	Pace         string `json:"pace"`
}

type uploadReportRequest struct {
	Title        string `json:"title"`
	Source       string `json:"source"`
	Participants string `json:"participants"`
	Owner        string `json:"owner"`
	Folder       string `json:"folder"`
}

type renameReportRequest struct {
	Title string `json:"title"`
}

type reportSearchResult struct {
	Section string `json:"section"`
	Time    string `json:"time,omitempty"`
	Title   string `json:"title,omitempty"`
	Text    string `json:"text"`
}

func (s *Server) reports(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}

	allReports := s.allReports()
	reports, total, limit, offset := filterReports(allReports, r, s.clock())
	writeJSON(w, http.StatusOK, reportsResponse{
		Reports: reports,
		Items:   reports,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
		Filters: buildReportFilters(allReports),
	})
}

func (s *Server) reportFilters(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}

	writeJSON(w, http.StatusOK, buildReportFilters(s.allReports()))
}

func (s *Server) reportUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}

	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		s.reportRecordingUpload(w, r)
		return
	}

	var req uploadReportRequest
	if r.Body != nil {
		_ = json.NewDecoder(http.MaxBytesReader(w, r.Body, 256*1024)).Decode(&req)
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = "Новая встреча"
	}
	owner := strings.TrimSpace(req.Owner)
	if owner == "" {
		owner = "Team AI"
	}
	source := strings.TrimSpace(req.Source)
	if source == "" {
		source = "Upload"
	}
	folder := strings.TrimSpace(req.Folder)
	if folder == "" {
		folder = "Обработка"
	}
	now := s.clock().UTC()
	initial := strings.ToUpper(firstRune(owner))
	if initial == "" {
		initial = "A"
	}

	report := reportRow{
		ID:               "uploaded-" + now.Format("20060102150405"),
		Title:            title,
		Source:           source,
		Type:             reportTypeFromSource(source),
		Date:             now.Format("02.01.2006"),
		Time:             now.Format("15:04"),
		Participants:     parsePositiveInt(req.Participants, 0),
		ParticipantNames: firstNonEmpty(req.Participants, "Ожидает анализа"),
		Score:            0,
		Folder:           folder,
		Owner:            owner,
		OwnerInitial:     initial,
		ThumbnailTone:    "mint",
		Week:             "Новые загрузки",
		Duration:         "00:00",
		Status:           "processing",
		ProcessingState:  "processing",
		CreatedAt:        now.Format(time.RFC3339),
		OccurredAt:       now,
	}
	s.storeReport(reportDetailResponse{
		Report: report,
		Summary: []summarySection{
			{Title: "Processing", Text: "Report accepted for AI processing."},
		},
		ActionItems:     []reportActionItem{},
		TranscriptLines: []reportTranscript{},
		Transcript:      []reportTranscript{},
		SpeakerStats:    []speakerStat{},
		Highlights:      []highlight{},
		Chapters:        []chapter{},
		AIQuestions: []string{
			"What is this meeting about?",
			"What action items were found?",
		},
	})

	writeJSON(w, http.StatusAccepted, map[string]any{
		"report":  report,
		"message": "Report accepted for AI processing",
	})
}

func (s *Server) reportRecordingUpload(w http.ResponseWriter, r *http.Request) {
	if s.stt == nil || !s.stt.Configured() {
		writeError(w, http.StatusServiceUnavailable, "Speech-to-text backend is not configured")
		return
	}

	input, err := s.readRecordingUploadInput(w, r, strings.TrimSpace(r.URL.Query().Get("roomName")))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	now := s.clock().UTC()
	owner := strings.TrimSpace(firstNonEmpty(input.Owner, "Team AI"))
	source := strings.TrimSpace(firstNonEmpty(input.Source, "Upload"))
	folder := strings.TrimSpace(firstNonEmpty(input.Folder, "Processed"))
	title := strings.TrimSpace(firstNonEmpty(input.Title, input.RoomName, "Uploaded meeting"))
	initial := strings.ToUpper(firstRune(owner))
	if initial == "" {
		initial = "A"
	}

	report := reportRow{
		ID:               fmt.Sprintf("uploaded-%s-%09d", now.Format("20060102150405"), now.Nanosecond()),
		Title:            title,
		Source:           source,
		Type:             reportTypeFromSource(source),
		Date:             now.Format("02.01.2006"),
		Time:             now.Format("15:04"),
		Participants:     parsePositiveInt(input.Participants, 0),
		ParticipantNames: firstNonEmpty(input.ParticipantNames, owner),
		Score:            0,
		Folder:           folder,
		Owner:            owner,
		OwnerInitial:     initial,
		ThumbnailTone:    "mint",
		Week:             "Uploads",
		Duration:         "00:00",
		Status:           "processing",
		ProcessingState:  "processing",
		CreatedAt:        now.Format(time.RFC3339),
		OccurredAt:       now,
	}

	recordingFile, recordingType, err := s.saveUploadedRecording(report.ID, input)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not save uploaded recording")
		return
	}

	detail := processingReportDetail(report, input.RoomName)
	if recordingFile != "" {
		detail.RecordingFile = recordingFile
		detail.RecordingType = recordingType
		detail.RecordingURL = "/api/reports/" + report.ID + "/recording/stream"
	}
	s.storeReport(detail)
	go s.processUploadedRecording(report.ID, input)

	writeJSON(w, http.StatusAccepted, map[string]any{
		"report":  report,
		"detail":  detail,
		"status":  "processing",
		"message": "Видео загружено. STT и AI-анализ выполняются в фоне.",
	})
}

func (s *Server) saveUploadedRecording(reportID string, input recordingUploadInput) (string, string, error) {
	if strings.TrimSpace(s.cfg.RecordingsStoragePath) == "" {
		return "", "", nil
	}

	extension := strings.ToLower(filepath.Ext(input.FileName))
	switch extension {
	case ".mp4", ".webm", ".mov", ".m4v", ".ogg", ".ogv", ".mp3", ".m4a", ".wav":
	default:
		switch {
		case strings.Contains(input.ContentType, "webm"):
			extension = ".webm"
		case strings.Contains(input.ContentType, "ogg"):
			extension = ".ogg"
		case strings.Contains(input.ContentType, "mpeg"):
			extension = ".mp3"
		case strings.Contains(input.ContentType, "wav"):
			extension = ".wav"
		default:
			extension = ".mp4"
		}
	}

	fileName := reportID + extension
	if err := os.MkdirAll(s.cfg.RecordingsStoragePath, 0o755); err != nil {
		return "", "", err
	}
	path := filepath.Join(s.cfg.RecordingsStoragePath, fileName)
	if err := os.WriteFile(path, input.Data, 0o600); err != nil {
		return "", "", err
	}

	contentType := strings.TrimSpace(input.ContentType)
	if contentType == "" {
		contentType = recordingContentType(fileName)
	}
	return fileName, contentType, nil
}

func recordingContentType(fileName string) string {
	switch strings.ToLower(filepath.Ext(fileName)) {
	case ".mp4", ".m4v", ".mov":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	case ".ogg", ".ogv":
		return "video/ogg"
	case ".mp3":
		return "audio/mpeg"
	case ".m4a":
		return "audio/mp4"
	case ".wav":
		return "audio/wav"
	default:
		return "application/octet-stream"
	}
}

func processingReportDetail(report reportRow, roomName string) reportDetailResponse {
	return reportDetailResponse{
		Report: report,
		Summary: []summarySection{
			{Title: "Обработка встречи", Text: "Видео загружено. Backend выполняет speech-to-text и AI-анализ в фоне."},
		},
		ActionItems:     []reportActionItem{},
		TranscriptLines: []reportTranscript{},
		Transcript:      []reportTranscript{},
		SpeakerStats:    []speakerStat{},
		Highlights:      []highlight{},
		Chapters:        []chapter{},
		AIQuestions: []string{
			"Когда будет готово резюме?",
			"Какие задачи появились после встречи?",
		},
		RoomName: roomName,
	}
}

func (s *Server) processUploadedRecording(reportID string, input recordingUploadInput) {
	timeout := s.cfg.STTTimeout + s.cfg.LLMTimeout + time.Minute
	if timeout <= time.Minute {
		timeout = 20 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	transcription, err := s.transcribeRecording(ctx, input)
	if err != nil {
		s.markUploadedReportFailed(reportID, err)
		return
	}

	transcriptText := truncateRunes(strings.TrimSpace(transcription.Text), maxTranscriptRunes)
	if transcriptText == "" {
		s.markUploadedReportFailed(reportID, errors.New("transcript is empty"))
		return
	}

	lines := transcriptLinesFromTranscription(transcription)
	if len(lines) == 0 {
		lines = transcriptLinesFromText(transcriptText)
	}

	analysis, err := s.generateMeetingAnalysisFromTranscript(ctx, input.RoomName, transcriptText, lines)
	if err != nil {
		analysis = fallbackAnalysisFromTranscript(input.RoomName, transcriptText, lines, s.clock())
	}

	detail, ok := s.reportDetailByID(reportID)
	if !ok {
		return
	}
	report := detail.Report
	report.Score = 90
	report.Duration = formatTranscriptTime(float64(max(30, len(lines)*30)))
	report.Status = "ready"
	report.ProcessingState = "ready"

	readyDetail := reportDetailFromAnalysis(report, analysis)
	readyDetail.RecordingURL = detail.RecordingURL
	readyDetail.RecordingFile = detail.RecordingFile
	readyDetail.RecordingType = detail.RecordingType
	s.storeReport(readyDetail)
}

func (s *Server) markUploadedReportFailed(reportID string, err error) {
	detail, ok := s.reportDetailByID(reportID)
	if !ok {
		return
	}

	detail.Report.Status = "failed"
	detail.Report.ProcessingState = "failed"
	detail.Report.Score = 0
	detail.Summary = []summarySection{
		{Title: "Ошибка обработки", Text: reportProcessingErrorMessage(err)},
	}
	detail.ActionItems = []reportActionItem{}
	detail.TranscriptLines = []reportTranscript{}
	detail.Transcript = []reportTranscript{}
	detail.SpeakerStats = []speakerStat{}
	detail.Highlights = []highlight{}
	detail.Chapters = []chapter{}

	s.storeReport(detail)
}

func reportProcessingErrorMessage(err error) string {
	if err == nil {
		return "STT service error: unknown error"
	}

	var apiErr *llm.APIError
	if errors.As(err, &apiErr) {
		return "STT service error: " + apiErr.Message
	}
	return "STT service error: " + err.Error()
}

func reportDetailFromAnalysis(report reportRow, analysis meetingAnalysis) reportDetailResponse {
	actionItems := make([]reportActionItem, 0, len(analysis.ActionItems))
	for i, item := range analysis.ActionItems {
		id := fmt.Sprintf("action-%d", i+1)
		actionItems = append(actionItems, reportActionItem{
			ID:       id,
			Title:    item.Task,
			Task:     item.Task,
			Owner:    item.Owner,
			Due:      item.Due,
			Priority: item.Priority,
			Status:   "open",
		})
	}

	transcript := make([]reportTranscript, 0, len(analysis.Transcript))
	for i, line := range analysis.Transcript {
		transcript = append(transcript, reportTranscript{
			ID:      fmt.Sprintf("t%d", i+1),
			Time:    line.Time,
			Speaker: line.Speaker,
			Text:    line.Text,
		})
	}

	speakerStats := make([]speakerStat, 0, len(analysis.Insights.TalkTime))
	for _, metric := range analysis.Insights.TalkTime {
		speakerStats = append(speakerStats, speakerStat{
			Name:         metric.Label,
			Role:         "Speaker",
			TalkTime:     metric.Value,
			TalkTimeText: fmt.Sprintf("%d%s", metric.Value, metric.Unit),
			Talk:         metric.Value,
			Sentiment:    analysis.Insights.Sentiment,
			Pace:         "",
		})
	}

	return reportDetailResponse{
		Report:          report,
		Summary:         analysis.Summary,
		ActionItems:     actionItems,
		TranscriptLines: transcript,
		Transcript:      transcript,
		SpeakerStats:    speakerStats,
		Highlights:      analysis.Highlights,
		Chapters:        analysis.Chapters,
		AIQuestions: []string{
			"Какие решения приняли на встрече?",
			"Какие задачи появились после встречи?",
			"Кратко перескажи встречу на русском.",
		},
		RoomName: analysis.RoomName,
	}
}

func (s *Server) storeReport(detail reportDetailResponse) {
	if detail.Report.ID == "" {
		return
	}
	s.reportsMu.Lock()
	defer s.reportsMu.Unlock()

	replaced := false
	for i, row := range s.generatedReports {
		if row.ID == detail.Report.ID {
			s.generatedReports[i] = detail.Report
			replaced = true
			break
		}
	}
	if !replaced {
		s.generatedReports = append([]reportRow{detail.Report}, s.generatedReports...)
	}
	s.generatedReportStore[detail.Report.ID] = detail
	if detail.RoomName != "" {
		s.latestRoomReports[detail.RoomName] = detail.Report.ID
	}
	s.saveReportsLocked()
}

func (s *Server) allReports() []reportRow {
	rows := demoReports()
	s.reportsMu.Lock()
	defer s.reportsMu.Unlock()
	if len(s.generatedReports) == 0 {
		return rows
	}
	out := make([]reportRow, 0, len(s.generatedReports)+len(rows))
	out = append(out, s.generatedReports...)
	out = append(out, rows...)
	return out
}

func (s *Server) reportDetailByID(id string) (reportDetailResponse, bool) {
	s.reportsMu.Lock()
	if detail, ok := s.generatedReportStore[id]; ok {
		s.reportsMu.Unlock()
		return detail, true
	}
	s.reportsMu.Unlock()
	return demoReportDetail(id, s.clock())
}

func (s *Server) reportByID(w http.ResponseWriter, r *http.Request) {
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/reports/"), "/")
	if rest == "" {
		writeError(w, http.StatusNotFound, "Report not found")
		return
	}

	parts := strings.Split(rest, "/")
	id := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}
	subAction := ""
	if len(parts) > 2 {
		subAction = parts[2]
	}

	detail, ok := s.reportDetailByID(id)
	if !ok {
		writeError(w, http.StatusNotFound, "Report not found")
		return
	}

	switch action {
	case "":
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, detail)
		case http.MethodPatch:
			s.renameReport(w, r, detail)
		case http.MethodDelete:
			writeJSON(w, http.StatusOK, map[string]string{
				"id":      detail.Report.ID,
				"status":  "deleted",
				"message": "Report deleted",
			})
		default:
			methodNotAllowed(w, http.MethodGet+", "+http.MethodPatch+", "+http.MethodDelete)
		}
	case "analysis":
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}
		writeJSON(w, http.StatusOK, reportDetailToMeetingAnalysis(detail, s.clock()))
	case "actions":
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}
		writeJSON(w, http.StatusOK, reportActions(detail.Report.ID))
	case "recording":
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}
		if subAction == "stream" {
			s.streamReportRecording(w, r, detail)
			return
		}
		recordingURL := firstNonEmpty(detail.RecordingURL, "/api/reports/"+detail.Report.ID+"/recording/stream")
		writeJSON(w, http.StatusOK, map[string]any{
			"reportId": detail.Report.ID,
			"duration": detail.Report.Duration,
			"url":      recordingURL,
			"markers":  []string{"00:42", "04:18", "12:05"},
		})
	case "tabs":
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}
		writeJSON(w, http.StatusOK, reportTabsPayload())
	case "notes":
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, map[string]any{"reportId": detail.Report.ID, "summary": detail.Summary, "actionItems": detail.ActionItems})
		case http.MethodPatch:
			var payload struct {
				Summary     []summarySection   `json:"summary"`
				ActionItems []reportActionItem `json:"actionItems"`
			}
			decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 128*1024))
			if err := decoder.Decode(&payload); err != nil {
				writeError(w, http.StatusBadRequest, "Invalid JSON body")
				return
			}
			if payload.Summary == nil {
				payload.Summary = detail.Summary
			}
			if payload.ActionItems == nil {
				payload.ActionItems = detail.ActionItems
			}
			writeJSON(w, http.StatusOK, map[string]any{
				"reportId":     detail.Report.ID,
				"summary":      payload.Summary,
				"actionItems":  payload.ActionItems,
				"status":       "saved",
				"message":      "Report notes saved",
				"updatedAt":    s.clock().UTC().Format(time.RFC3339),
				"editEndpoint": "/api/reports/" + detail.Report.ID + "/notes",
			})
		default:
			methodNotAllowed(w, http.MethodGet+", "+http.MethodPatch)
		}
	case "action-items":
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"reportId": detail.Report.ID, "items": detail.ActionItems})
	case "transcript":
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"reportId": detail.Report.ID, "lines": detail.TranscriptLines})
	case "deep-dive":
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"reportId": detail.Report.ID, "speakerStats": detail.SpeakerStats})
	case "highlights":
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"reportId": detail.Report.ID, "items": detail.Highlights})
	case "chapters":
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"reportId": detail.Report.ID, "items": detail.Chapters})
	case "download":
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}
		s.downloadReport(w, r, detail)
	case "share":
		if r.Method != http.MethodPost {
			methodNotAllowed(w, http.MethodPost)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{
			"id":    detail.Report.ID,
			"title": detail.Report.Title,
			"url":   "/report/" + detail.Report.ID,
		})
	case "send":
		if r.Method != http.MethodPost {
			methodNotAllowed(w, http.MethodPost)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"id":      detail.Report.ID,
			"status":  "queued",
			"message": "Report send job queued",
		})
	case "copy":
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{
			"id":    detail.Report.ID,
			"title": detail.Report.Title,
			"text":  reportDetailContext(detail),
		})
	case "search":
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"query":   strings.TrimSpace(r.URL.Query().Get("q")),
			"results": searchReportDetail(detail, r.URL.Query().Get("q")),
		})
	case "history":
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"reportId": detail.Report.ID,
			"history":  []aiChatMessage{},
		})
	case "prompts":
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"reportId": detail.Report.ID,
			"prompts":  detail.AIQuestions,
		})
	case "chat":
		s.reportChat(w, r, detail)
	default:
		writeError(w, http.StatusNotFound, "Report action not found")
	}
}

func (s *Server) streamReportRecording(w http.ResponseWriter, r *http.Request, detail reportDetailResponse) {
	if detail.RecordingFile != "" {
		s.serveStoredRecording(w, r, detail, false)
		return
	}
	if detail.RecordingURL != "" {
		http.Redirect(w, r, detail.RecordingURL, http.StatusTemporaryRedirect)
		return
	}

	title := html.EscapeString(detail.Report.Title)
	duration := html.EscapeString(detail.Report.Duration)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `<!doctype html>
<html lang="ru">
<head><meta charset="utf-8"><title>%s</title></head>
<body style="font-family: system-ui, sans-serif; margin: 32px; background: #0f172a; color: white;">
<h1>%s</h1>
<p>Запись встречи пока хранится как backend-заглушка. Длительность: %s.</p>
<p>Когда появится реальное storage/CDN, этот endpoint можно заменить на redirect или video stream.</p>
</body>
</html>`, title, title, duration)
}

func (s *Server) serveStoredRecording(w http.ResponseWriter, r *http.Request, detail reportDetailResponse, attachment bool) {
	fileName := filepath.Base(detail.RecordingFile)
	if fileName == "." || fileName == string(filepath.Separator) || fileName == "" {
		writeError(w, http.StatusNotFound, "Видео встречи пока недоступно")
		return
	}

	path := filepath.Join(s.cfg.RecordingsStoragePath, fileName)
	file, err := os.Open(path)
	if err != nil {
		writeError(w, http.StatusNotFound, "Видео встречи пока недоступно")
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		writeError(w, http.StatusNotFound, "Видео встречи пока недоступно")
		return
	}

	contentType := firstNonEmpty(detail.RecordingType, recordingContentType(fileName))
	w.Header().Set("Content-Type", contentType)
	if attachment {
		downloadName := detail.Report.ID + filepath.Ext(fileName)
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", downloadName))
	}
	http.ServeContent(w, r, fileName, stat.ModTime(), file)
}

func (s *Server) renameReport(w http.ResponseWriter, r *http.Request, detail reportDetailResponse) {
	var req renameReportRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16*1024))
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	if utf8.RuneCountInString(title) > 160 {
		writeError(w, http.StatusBadRequest, "title is too long")
		return
	}

	detail.Report.Title = title
	writeJSON(w, http.StatusOK, map[string]any{
		"report": detail.Report,
		"status": "renamed",
	})
}

func (s *Server) reportChat(w http.ResponseWriter, r *http.Request, detail reportDetailResponse) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}

	var req aiChatRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64*1024))
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	message := strings.TrimSpace(req.Message)
	if message == "" {
		writeError(w, http.StatusBadRequest, "message is required")
		return
	}
	if utf8.RuneCountInString(message) > maxChatMessageRunes {
		writeError(w, http.StatusBadRequest, "message is too long")
		return
	}

	contextText := truncateRunes(firstNonEmpty(req.Context, reportDetailContext(detail)), maxChatContextRunes)
	if s.ai == nil || !s.ai.Configured() {
		writeJSON(w, http.StatusOK, aiChatResponse{Answer: fallbackReportAnswerSafe(detail, message)})
		return
	}

	messages := []llm.Message{
		{
			Role:    "system",
			Content: "Ты AI-помощник AlemLive. Отвечай кратко на русском языке, используя только контекст выбранного отчёта.",
		},
		{Role: "user", Content: "Контекст отчёта:\n" + contextText},
	}
	messages = append(messages, sanitizeChatHistory(req.History)...)
	messages = append(messages, llm.Message{Role: "user", Content: message})

	answer, err := s.ai.Chat(r.Context(), messages, llm.ChatOptions{
		Temperature: 0.3,
		MaxTokens:   700,
	})
	if err != nil {
		log.Printf("report chat AI fallback for report %s: %v", detail.Report.ID, err)
		writeJSON(w, http.StatusOK, aiChatResponse{Answer: fallbackReportAnswerSafe(detail, message)})
		return
	}

	writeJSON(w, http.StatusOK, aiChatResponse{Answer: answer})
}

func (s *Server) downloadReport(w http.ResponseWriter, r *http.Request, detail reportDetailResponse) {
	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	if format == "" {
		format = "summary"
	}

	switch format {
	case "summary":
		downloadReportSummary(w, detail)
	case "transcript":
		downloadReportTranscript(w, detail)
	case "trailer", "highlights", "video":
		s.downloadReportVideo(w, r, detail, format)
	default:
		writeError(w, http.StatusBadRequest, "Unsupported download format")
	}
}

func downloadReportSummary(w http.ResponseWriter, detail reportDetailResponse) {
	filename := detail.Report.ID + "-summary.txt"
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.WriteHeader(http.StatusOK)

	fmt.Fprintf(w, "%s\n\n", detail.Report.Title)
	for _, section := range detail.Summary {
		fmt.Fprintf(w, "%s\n%s\n\n", section.Title, section.Text)
	}
	fmt.Fprintln(w, "Action items")
	for _, item := range detail.ActionItems {
		fmt.Fprintf(w, "- %s (%s, %s)\n", item.Title, item.Owner, item.Due)
	}
}

func downloadReportTranscript(w http.ResponseWriter, detail reportDetailResponse) {
	filename := detail.Report.ID + "-transcript.txt"
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.WriteHeader(http.StatusOK)

	fmt.Fprintf(w, "%s\n\n", detail.Report.Title)
	for _, line := range detail.TranscriptLines {
		fmt.Fprintf(w, "%s %s: %s\n", line.Time, line.Speaker, line.Text)
	}
}

func (s *Server) downloadReportVideo(w http.ResponseWriter, r *http.Request, detail reportDetailResponse, format string) {
	if detail.RecordingFile != "" {
		s.serveStoredRecording(w, r, detail, true)
		return
	}

	if detail.RecordingURL == "" {
		writeError(w, http.StatusNotFound, "Видео встречи пока недоступно: запись появится после обработки LiveKit Egress")
		return
	}

	request, err := http.NewRequestWithContext(r.Context(), http.MethodGet, detail.RecordingURL, nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not create recording download request")
		return
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		writeError(w, http.StatusBadGateway, "Could not download meeting recording")
		return
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		writeError(w, http.StatusBadGateway, "Meeting recording storage returned an error")
		return
	}

	contentType := response.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "video/mp4"
	}
	filename := fmt.Sprintf("%s-%s.mp4", detail.Report.ID, format)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	if response.ContentLength > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(response.ContentLength, 10))
	}
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, response.Body)
}

func searchReportDetail(detail reportDetailResponse, query string) []reportSearchResult {
	query = strings.ToLower(strings.TrimSpace(query))
	results := make([]reportSearchResult, 0)
	add := func(section, timeValue, title, text string) {
		if query != "" && !strings.Contains(strings.ToLower(title+" "+text), query) {
			return
		}
		results = append(results, reportSearchResult{
			Section: section,
			Time:    timeValue,
			Title:   title,
			Text:    text,
		})
	}

	for _, item := range detail.Summary {
		add("summary", "", item.Title, item.Text)
	}
	for _, item := range detail.ActionItems {
		add("actionItems", "", item.Title, item.Owner+" / "+item.Due)
	}
	for _, line := range detail.TranscriptLines {
		add("transcript", line.Time, line.Speaker, line.Text)
	}
	for _, item := range detail.Highlights {
		add("highlights", item.Time, item.Title, firstNonEmpty(item.Note, item.Text))
	}
	for _, item := range detail.Chapters {
		add("chapters", firstNonEmpty(item.Time, item.Start), item.Title, item.Text)
	}

	return results
}

func reportActions(reportID string) []map[string]any {
	return []map[string]any{
		{"id": "share", "label": "Поделиться", "method": http.MethodPost, "endpoint": "/api/reports/" + reportID + "/share", "enabled": true},
		{"id": "download", "label": "Скачать", "method": http.MethodGet, "endpoint": "/api/reports/" + reportID + "/download", "enabled": true},
		{"id": "rename", "label": "Переименовать отчёт", "method": http.MethodPatch, "endpoint": "/api/reports/" + reportID, "enabled": true},
		{"id": "delete", "label": "Удалить отчёт", "method": http.MethodDelete, "endpoint": "/api/reports/" + reportID, "enabled": true, "danger": true},
		{"id": "send", "label": "Отправить в...", "method": http.MethodPost, "endpoint": "/api/reports/" + reportID + "/send", "enabled": true},
		{"id": "copy", "label": "Копировать отчёт", "method": http.MethodGet, "endpoint": "/api/reports/" + reportID + "/copy", "enabled": true},
	}
}

func reportTabsPayload() []map[string]string {
	return []map[string]string{
		{"id": "notes", "label": "Заметки", "endpoint": "notes"},
		{"id": "transcript", "label": "Транскрипт", "endpoint": "transcript"},
		{"id": "deepDive", "label": "Глубокое погружение", "endpoint": "deep-dive"},
		{"id": "highlights", "label": "Основные моменты", "endpoint": "highlights"},
		{"id": "chapters", "label": "Главы", "endpoint": "chapters"},
	}
}

func filterReports(rows []reportRow, r *http.Request, now time.Time) ([]reportRow, int, int, int) {
	query := r.URL.Query()
	search := strings.ToLower(strings.TrimSpace(firstNonEmpty(query.Get("q"), query.Get("search"))))
	source := strings.ToLower(strings.TrimSpace(query.Get("source")))
	folder := strings.ToLower(strings.TrimSpace(query.Get("folder")))
	owner := strings.ToLower(strings.TrimSpace(query.Get("owner")))
	mode := strings.ToLower(strings.TrimSpace(firstNonEmpty(query.Get("mode"), query.Get("status"))))
	types := parseReportTypes(query.Get("types"))
	from, to := reportDateRange(query.Get("datePreset"), query.Get("preset"), query.Get("timeFilter"), query.Get("timeFilterMode"), query.Get("from"), query.Get("to"), query.Get("dateFrom"), query.Get("dateTo"), now)

	filtered := make([]reportRow, 0, len(rows))
	for _, row := range rows {
		if search != "" && !reportContains(row, search) {
			continue
		}
		if source != "" && strings.ToLower(row.Source) != source {
			continue
		}
		if folder != "" && strings.ToLower(row.Folder) != folder {
			continue
		}
		if owner != "" && strings.ToLower(row.Owner) != owner {
			continue
		}
		if mode == "incomplete" && row.ProcessingState == "ready" && row.Score >= 90 {
			continue
		}
		if len(types) > 0 {
			if _, ok := types[reportTypeValue(row)]; !ok {
				continue
			}
		}
		if !from.IsZero() && row.OccurredAt.Before(from) {
			continue
		}
		if !to.IsZero() && row.OccurredAt.After(to) {
			continue
		}
		filtered = append(filtered, row)
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].OccurredAt.After(filtered[j].OccurredAt)
	})

	limit := parsePositiveInt(query.Get("limit"), len(filtered))
	offset := parsePositiveInt(query.Get("offset"), 0)
	if offset > len(filtered) {
		return []reportRow{}, len(filtered), limit, offset
	}
	end := len(filtered)
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}
	return filtered[offset:end], len(filtered), limit, offset
}

func reportContains(row reportRow, search string) bool {
	values := []string{
		row.Title,
		row.Source,
		row.Type,
		row.Date,
		row.Time,
		strconv.Itoa(row.Participants),
		row.ParticipantNames,
		row.Folder,
		row.Owner,
		row.Week,
		row.Status,
	}
	for _, value := range values {
		if strings.Contains(strings.ToLower(value), search) {
			return true
		}
	}
	return false
}

func parseReportTypes(value string) map[string]struct{} {
	types := map[string]struct{}{}
	for _, item := range strings.Split(value, ",") {
		normalized := normalizeReportType(item)
		if normalized == "" {
			continue
		}
		types[normalized] = struct{}{}
	}
	return types
}

func reportTypeValue(row reportRow) string {
	return firstNonEmpty(normalizeReportType(row.Type), reportTypeFromSource(row.Source))
}

func reportTypeFromSource(source string) string {
	return normalizeReportType(source)
}

func normalizeReportType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "all":
		return ""
	case "meeting", "meetings", "google meet", "zoom", "microsoft teams", "alemlive":
		return "meeting"
	case "readout":
		return "readout"
	case "daily":
		return "daily"
	case "upload":
		return "upload"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func reportDateRange(datePreset, preset, timeFilter, timeFilterMode, fromValue, toValue, dateFrom, dateTo string, now time.Time) (time.Time, time.Time) {
	preset = strings.ToLower(strings.TrimSpace(firstNonEmpty(datePreset, preset, timeFilter, timeFilterMode)))
	from := parseReportDate(firstNonEmpty(fromValue, dateFrom))
	to := parseReportDate(firstNonEmpty(toValue, dateTo))
	if !from.IsZero() || !to.IsZero() {
		return startOfDay(from), endOfDay(to)
	}

	today := startOfDay(now)
	switch preset {
	case "", "all":
		return time.Time{}, time.Time{}
	case "today":
		return today, endOfDay(today)
	case "last7":
		return today.AddDate(0, 0, -6), endOfDay(today)
	case "last30":
		return today.AddDate(0, 0, -29), endOfDay(today)
	case "last90":
		return today.AddDate(0, 0, -89), endOfDay(today)
	case "last6months":
		return today.AddDate(0, -6, 0), endOfDay(today)
	case "last12months":
		return today.AddDate(-1, 0, 0), endOfDay(today)
	default:
		return time.Time{}, time.Time{}
	}
}

func parseReportDate(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	for _, layout := range []string{"2006-01-02", time.RFC3339, "02.01.2006"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed
		}
	}
	return time.Time{}
}

func startOfDay(value time.Time) time.Time {
	if value.IsZero() {
		return time.Time{}
	}
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
}

func endOfDay(value time.Time) time.Time {
	if value.IsZero() {
		return time.Time{}
	}
	return startOfDay(value).Add(24*time.Hour - time.Nanosecond)
}

func parsePositiveInt(value string, fallback int) int {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return fallback
	}
	return parsed
}

func buildReportFilters(rows []reportRow) reportFilters {
	return reportFilters{
		Sources: uniqueReportValues(rows, func(row reportRow) string { return row.Source }),
		Folders: uniqueReportValues(rows, func(row reportRow) string {
			return row.Folder
		}),
		Owners: uniqueReportValues(rows, func(row reportRow) string { return row.Owner }),
		QuickDateOptions: []quickDateOption{
			{ID: "all", Label: "Все время"},
			{ID: "today", Label: "Сегодня"},
			{ID: "last7", Label: "Последние 7 дней"},
			{ID: "last30", Label: "Последние 30 дней"},
			{ID: "last90", Label: "Последние 90 дней"},
			{ID: "last6months", Label: "Последние 6 месяцев"},
			{ID: "last12months", Label: "Последние 12 месяцев"},
		},
	}
}

func uniqueReportValues(rows []reportRow, pick func(reportRow) string) []string {
	seen := map[string]struct{}{}
	values := make([]string, 0)
	for _, row := range rows {
		value := strings.TrimSpace(pick(row))
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		values = append(values, value)
	}
	sort.Strings(values)
	return values
}

func demoReports() []reportRow {
	return []reportRow{
		newDemoReport("read-intro", "Ввод в Alem AI - Пример отчёта", "Google Meet", "2026-01-02T02:00:00Z", "02:00 - 03:45", 4, "Alison Barker, Мади, Айдана, +1 больше", 89, "Образцы отчётов", "Мади", "М", "teal", "НЕДЕЛЯ С 29 ДЕК.-4 ЯНВ., 2025", "01:45", "needs_review"),
		newDemoReport("meeting-usage", "Использование отчёта собрания - Пример отчёта", "Google Meet", "2026-01-02T01:00:00Z", "01:00 - 01:04", 4, "Alison Barker, Мади, Айдана, +1 больше", 89, "Образцы отчётов", "Айдана", "А", "blue", "НЕДЕЛЯ С 29 ДЕК.-4 ЯНВ., 2025", "00:04", "needs_review"),
		newDemoReport("copilot-search", "Используйте Copilot для поиска - Пример отчёта", "Google Meet", "2026-01-02T00:00:00Z", "00:00 - 00:07", 4, "Alison Barker, Мади, Айдана, +1 больше", 88, "Образцы отчётов", "Елиас", "Е", "violet", "НЕДЕЛЯ С 29 ДЕК.-4 ЯНВ., 2025", "00:07", "needs_review"),
		newDemoReport("mobile-guide", "Руководство по использованию настольного и мобильного приложения", "Google Meet", "2026-01-01T23:00:00Z", "23:00 - 23:04", 5, "Alison Barker, Мади, Айдана, Kelsey, +1 больше", 92, "Образцы отчётов", "Келси", "К", "green", "НЕДЕЛЯ С 29 ДЕК.-4 ЯНВ., 2025", "00:04", "ready"),
		newDemoReport("real-cases", "Исследуйте реальные случаи использования - Пример отчёта", "Google Meet", "2026-01-01T22:00:00Z", "22:00 - 22:08", 4, "Alison Barker, Мади, Айдана, Сара", 87, "Образцы отчётов", "Сара", "С", "rose", "НЕДЕЛЯ С 29 ДЕК.-4 ЯНВ., 2025", "00:08", "needs_review"),
	}
}

func newDemoReport(id, title, source, occurredAt, meetingTime string, participants int, participantNames string, score int, folder, owner, ownerInitial, tone, week, duration, state string) reportRow {
	parsed, _ := time.Parse(time.RFC3339, occurredAt)
	return reportRow{
		ID:               id,
		Title:            title,
		Source:           source,
		Type:             reportTypeFromSource(source),
		Date:             parsed.Format("02.01.2006"),
		Time:             meetingTime,
		Participants:     participants,
		ParticipantNames: participantNames,
		Score:            score,
		Folder:           folder,
		Owner:            owner,
		OwnerInitial:     ownerInitial,
		ThumbnailTone:    tone,
		Week:             week,
		Duration:         duration,
		Status:           state,
		ProcessingState:  state,
		CreatedAt:        parsed.UTC().Format(time.RFC3339),
		OccurredAt:       parsed,
	}
}

func demoReportDetail(id string, now time.Time) (reportDetailResponse, bool) {
	var report reportRow
	found := false
	for _, row := range demoReports() {
		if row.ID == id {
			report = row
			found = true
			break
		}
	}
	if !found {
		return reportDetailResponse{}, false
	}

	summary := []summarySection{
		{Title: "Что обсудили", Text: "Команда согласовала структуру AI отчёта после встречи: резюме, задачи, транскрипт, метрики, основные моменты и главы."},
		{Title: "Решения", Text: "Backend должен отдавать готовый контракт для списка отчётов, фильтров и детальной страницы, чтобы frontend мог подключиться без хранения моков."},
		{Title: "Следующие шаги", Text: "Подключить запись встречи, сохранить artifacts и отправить transcript в LLM для финального анализа."},
	}
	actionItems := []reportActionItem{
		{ID: "questions", Title: "Подготовить список вопросов для демо клиента", Task: "Подготовить список вопросов для демо клиента", Owner: "Мади Орысбек", Due: "Сегодня, 18:00", Priority: "High", Status: "open"},
		{ID: "livekit-token", Title: "Проверить backend endpoint для LiveKit token", Task: "Проверить backend endpoint для LiveKit token", Owner: "Айдана Сейт", Due: "Завтра, 11:00", Priority: "High", Status: "open"},
		{ID: "report-ui", Title: "Обновить UI отчёта после тестовой встречи", Task: "Обновить UI отчёта после тестовой встречи", Owner: "Team AI", Due: "После созвона", Priority: "Medium", Status: "open"},
	}
	transcript := []reportTranscript{
		{ID: "t1", Time: "00:00", Speaker: "Alison Barker", Text: "Давайте зафиксируем, какие блоки должен показывать отчёт после встречи."},
		{ID: "t2", Time: "04:12", Speaker: "Мади Орысбек", Text: "Пользователь создаёт комнату, присоединяется по названию, а URL и token приходят через backend."},
		{ID: "t3", Time: "09:45", Speaker: "Айдана Сейт", Text: "После созвона агент должен показать резюме, задачи, транскрипт, метрики и главы."},
		{ID: "t4", Time: "18:20", Speaker: "Team AI", Text: "Чат справа должен отвечать по контексту выбранного отчёта."},
	}

	return reportDetailResponse{
		Report:          report,
		Summary:         summary,
		ActionItems:     actionItems,
		TranscriptLines: transcript,
		Transcript:      transcript,
		SpeakerStats: []speakerStat{
			{Name: "Мади", Role: "Product", TalkTime: 48, TalkTimeText: "48%", Talk: 48, Sentiment: "Позитивный", Pace: "142 слов/мин"},
			{Name: "Айдана", Role: "Backend", TalkTime: 34, TalkTimeText: "34%", Talk: 34, Sentiment: "Нейтральный", Pace: "128 слов/мин"},
			{Name: "Team AI", Role: "AI", TalkTime: 18, TalkTimeText: "18%", Talk: 18, Sentiment: "Фокус", Pace: "96 слов/мин"},
		},
		Highlights: []highlight{
			{Time: "03:20", Title: "Решение по входу в комнату", Text: "Название комнаты становится главным способом подключения.", Note: "Название комнаты становится главным способом подключения.", Type: "Decision"},
			{Time: "17:45", Title: "Риск по backend", Text: "Если backend не запущен, агент должен явно показать ошибку подключения.", Note: "Если backend не запущен, агент должен явно показать ошибку подключения.", Type: "Risk"},
			{Time: "28:10", Title: "Следующий шаг", Text: "Добавить автоматический отчёт после завершения митинга.", Note: "Добавить автоматический отчёт после завершения митинга.", Type: "Action"},
		},
		Chapters: []chapter{
			{Start: "00:00", End: "04:00", Time: "00:00", Title: "Старт и цель встречи", Text: "Цель встречи и структура будущего AI отчёта.", Duration: "4 мин"},
			{Start: "04:01", End: "13:09", Time: "04:01", Title: "LiveKit вход и комнаты", Text: "Создание комнаты и получение токена через backend.", Duration: "9 мин"},
			{Start: "13:10", End: "25:29", Time: "13:10", Title: "Структура AI отчёта", Text: "Резюме, задачи, транскрипт, метрики и highlights.", Duration: "12 мин"},
			{Start: "25:30", End: "32:30", Time: "25:30", Title: "Action items и финальные решения", Text: "Вопросы к AI по контексту отчёта.", Duration: "7 мин"},
		},
		AIQuestions: []string{
			"Какие решения приняли на встрече?",
			"Какие задачи появились после встречи?",
			"Переведите резюме встречи на русский.",
		},
	}, true
}

func reportDetailToMeetingAnalysis(detail reportDetailResponse, now time.Time) meetingAnalysis {
	talkTime := make([]metricValue, 0, len(detail.SpeakerStats))
	for _, speaker := range detail.SpeakerStats {
		talkTime = append(talkTime, metricValue{Label: speaker.Name, Value: speaker.TalkTime, Unit: "%"})
	}
	transcript := make([]transcriptLine, 0, len(detail.TranscriptLines))
	for _, line := range detail.TranscriptLines {
		transcript = append(transcript, transcriptLine{Time: line.Time, Speaker: line.Speaker, Text: line.Text})
	}
	actions := make([]actionItem, 0, len(detail.ActionItems))
	for _, item := range detail.ActionItems {
		actions = append(actions, actionItem{Task: item.Title, Owner: item.Owner, Due: item.Due, Priority: item.Priority})
	}

	return meetingAnalysis{
		RoomName:    detail.Report.ID,
		GeneratedAt: now.UTC().Format(time.RFC3339),
		Summary:     detail.Summary,
		ActionItems: actions,
		Transcript:  transcript,
		Insights: meetingInsights{
			Sentiment: "Constructive",
			TalkTime:  talkTime,
			SpeechRate: []metricValue{
				{Label: "Average", Value: 132, Unit: "wpm"},
			},
			Interruptions: []metricValue{
				{Label: "Total", Value: 2, Unit: "times"},
			},
			Engagement: []metricValue{
				{Label: "Questions", Value: len(detail.AIQuestions), Unit: "items"},
				{Label: "Action items", Value: len(detail.ActionItems), Unit: "items"},
			},
		},
		Highlights: detail.Highlights,
		Chapters:   detail.Chapters,
	}
}

func reportDetailContext(detail reportDetailResponse) string {
	var b strings.Builder
	b.WriteString("Отчёт: ")
	b.WriteString(detail.Report.Title)
	b.WriteString("\nID: ")
	b.WriteString(detail.Report.ID)
	b.WriteString("\nИсточник: ")
	b.WriteString(detail.Report.Source)
	b.WriteString("\nДата: ")
	b.WriteString(detail.Report.Date)
	b.WriteString(" ")
	b.WriteString(detail.Report.Time)
	b.WriteString("\nУчастники: ")
	b.WriteString(firstNonEmpty(detail.Report.ParticipantNames, strconv.Itoa(detail.Report.Participants)))
	b.WriteString("\n\nSummary:\n")
	for _, section := range detail.Summary {
		b.WriteString("- ")
		b.WriteString(section.Title)
		b.WriteString(": ")
		b.WriteString(section.Text)
		b.WriteString("\n")
	}
	b.WriteString("\nAction items:\n")
	for _, item := range detail.ActionItems {
		b.WriteString("- ")
		b.WriteString(item.Title)
		b.WriteString(" / ")
		b.WriteString(item.Owner)
		b.WriteString(" / ")
		b.WriteString(item.Due)
		b.WriteString("\n")
	}
	b.WriteString("\nTranscript:\n")
	for _, line := range detail.TranscriptLines {
		b.WriteString(line.Time)
		b.WriteString(" ")
		b.WriteString(line.Speaker)
		b.WriteString(": ")
		b.WriteString(line.Text)
		b.WriteString("\n")
	}
	return b.String()
}

func fallbackReportAnswer(detail reportDetailResponse, message string) string {
	message = strings.ToLower(message)
	if strings.Contains(message, "зада") || strings.Contains(message, "action") || strings.Contains(message, "todo") {
		var b strings.Builder
		b.WriteString("Задачи после встречи:\n")
		for _, item := range detail.ActionItems {
			b.WriteString("- ")
			b.WriteString(item.Title)
			b.WriteString(" — ")
			b.WriteString(item.Owner)
			b.WriteString(", ")
			b.WriteString(item.Due)
			b.WriteString("\n")
		}
		return strings.TrimSpace(b.String())
	}
	if strings.Contains(message, "решен") || strings.Contains(message, "decision") {
		var b strings.Builder
		b.WriteString("Ключевые решения:\n")
		for _, item := range detail.Highlights {
			if strings.EqualFold(item.Type, "Decision") {
				b.WriteString("- ")
				b.WriteString(item.Title)
				b.WriteString(": ")
				b.WriteString(item.Text)
				b.WriteString("\n")
			}
		}
		return strings.TrimSpace(b.String())
	}
	return "Короткое резюме: " + detail.Summary[0].Text
}

func fallbackReportAnswerSafe(detail reportDetailResponse, message string) string {
	message = strings.ToLower(message)
	if strings.Contains(message, "зада") || strings.Contains(message, "action") || strings.Contains(message, "todo") {
		if len(detail.ActionItems) == 0 {
			return "В отчете пока нет найденных задач."
		}
		var b strings.Builder
		b.WriteString("Задачи после встречи:\n")
		for _, item := range detail.ActionItems {
			b.WriteString("- ")
			b.WriteString(firstNonEmpty(item.Title, item.Task, "Задача"))
			if item.Owner != "" {
				b.WriteString(" — ")
				b.WriteString(item.Owner)
			}
			if item.Due != "" {
				b.WriteString(", ")
				b.WriteString(item.Due)
			}
			b.WriteString("\n")
		}
		return strings.TrimSpace(b.String())
	}
	if strings.Contains(message, "решен") || strings.Contains(message, "decision") {
		var b strings.Builder
		for _, item := range detail.Highlights {
			if strings.EqualFold(item.Type, "Decision") {
				b.WriteString("- ")
				b.WriteString(item.Title)
				if item.Text != "" {
					b.WriteString(": ")
					b.WriteString(item.Text)
				}
				b.WriteString("\n")
			}
		}
		if strings.TrimSpace(b.String()) == "" {
			return "В отчете пока нет явно выделенных решений."
		}
		return "Ключевые решения:\n" + strings.TrimSpace(b.String())
	}
	if len(detail.Summary) == 0 {
		return "Отчет еще обрабатывается или не содержит сводку."
	}
	return "Короткое резюме: " + detail.Summary[0].Text
}

func firstRune(value string) string {
	for _, r := range strings.TrimSpace(value) {
		return string(r)
	}
	return ""
}
