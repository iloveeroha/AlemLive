package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
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

	reports, total, limit, offset := filterReports(demoReports(), r, s.clock())
	writeJSON(w, http.StatusOK, reportsResponse{
		Reports: reports,
		Items:   reports,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
		Filters: buildReportFilters(demoReports()),
	})
}

func (s *Server) reportFilters(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}

	writeJSON(w, http.StatusOK, buildReportFilters(demoReports()))
}

func (s *Server) reportUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
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

	writeJSON(w, http.StatusAccepted, map[string]any{
		"report":  report,
		"message": "Report accepted for AI processing",
	})
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

	detail, ok := demoReportDetail(id, s.clock())
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
	case "download":
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}
		s.downloadReport(w, detail)
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
		writeJSON(w, http.StatusOK, aiChatResponse{Answer: fallbackReportAnswer(detail, message)})
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
		writeError(w, http.StatusBadGateway, "AI service is unavailable")
		return
	}

	writeJSON(w, http.StatusOK, aiChatResponse{Answer: answer})
}

func (s *Server) downloadReport(w http.ResponseWriter, detail reportDetailResponse) {
	filename := detail.Report.ID + ".txt"
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

func filterReports(rows []reportRow, r *http.Request, now time.Time) ([]reportRow, int, int, int) {
	query := r.URL.Query()
	search := strings.ToLower(strings.TrimSpace(firstNonEmpty(query.Get("q"), query.Get("search"))))
	source := strings.ToLower(strings.TrimSpace(query.Get("source")))
	folder := strings.ToLower(strings.TrimSpace(query.Get("folder")))
	owner := strings.ToLower(strings.TrimSpace(query.Get("owner")))
	mode := strings.ToLower(strings.TrimSpace(firstNonEmpty(query.Get("mode"), query.Get("status"))))
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

func firstRune(value string) string {
	for _, r := range strings.TrimSpace(value) {
		return string(r)
	}
	return ""
}
