package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/iloveeroha/AlemLive/backend/internal/llm"
)

const (
	maxChatMessageRunes = 4000
	maxChatContextRunes = 12000
	maxChatHistoryItems = 8
)

type aiChatRequest struct {
	Message  string          `json:"message"`
	ReportID string          `json:"reportId"`
	Context  string          `json:"context"`
	History  []aiChatMessage `json:"history"`
}

type aiChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type aiChatResponse struct {
	Answer string `json:"answer"`
}

type aiStatusResponse struct {
	Configured            bool   `json:"configured"`
	BaseURL               string `json:"baseUrl,omitempty"`
	Model                 string `json:"model,omitempty"`
	STTConfigured         bool   `json:"sttConfigured"`
	STTBaseURL            string `json:"sttBaseUrl,omitempty"`
	STTModel              string `json:"sttModel,omitempty"`
	DiarizationConfigured bool   `json:"diarizationConfigured"`
	DiarizationBaseURL    string `json:"diarizationBaseUrl,omitempty"`
}

func (s *Server) aiStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}

	response := aiStatusResponse{}
	if s.ai != nil && s.ai.Configured() {
		response.Configured = true
		response.BaseURL = s.cfg.LLMBaseURL
		response.Model = s.cfg.LLMModel
	}
	if s.stt != nil && s.stt.Configured() {
		response.STTConfigured = true
		response.STTBaseURL = s.cfg.STTBaseURL
		response.STTModel = s.cfg.STTModel
	}
	if s.diarizer != nil && s.diarizer.Configured() {
		response.DiarizationConfigured = true
		response.DiarizationBaseURL = s.cfg.DiarizationBaseURL
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) aiChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	if s.ai == nil || !s.ai.Configured() {
		writeError(w, http.StatusServiceUnavailable, "AI backend is not configured")
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

	contextText := strings.TrimSpace(req.Context)
	if contextText == "" {
		contextText = defaultReportContext(req.ReportID, s.clock)
	}
	contextText = truncateRunes(contextText, maxChatContextRunes)

	messages := []llm.Message{
		{
			Role:    "system",
			Content: "Ты AI-помощник AlemLive. Отвечай кратко и полезно на русском языке. Используй контекст отчета о встрече, если он передан. Не выдумывай факты, которых нет в контексте.",
		},
	}
	messages = append(messages, llm.Message{
		Role:    "system",
		Content: "Отвечай обычным текстом на русском языке. Не используй Markdown: без **, без #, без списков с маркерами и без code fences.",
	})
	if contextText != "" {
		messages = append(messages, llm.Message{
			Role:    "user",
			Content: "Контекст отчета о встрече:\n" + contextText,
		})
	}
	messages = []llm.Message{
		{Role: "system", Content: "Ты AI-помощник AlemLive. Отвечай кратко и полезно на русском языке. Используй контекст отчёта о встрече, если он передан. Не выдумывай факты, которых нет в контексте."},
		{Role: "system", Content: "Отвечай обычным текстом на русском языке. Не используй Markdown: без **, без #, без списков с маркерами и без code fences."},
	}
	if contextText != "" {
		messages = append(messages, llm.Message{
			Role:    "user",
			Content: "Контекст отчёта о встрече:\n" + contextText,
		})
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

	writeJSON(w, http.StatusOK, aiChatResponse{Answer: plainTextAIAnswer(answer)})
}

func plainTextAIAnswer(value string) string {
	value = strings.TrimSpace(value)
	replacer := strings.NewReplacer(
		"**", "",
		"__", "",
		"### ", "",
		"## ", "",
		"# ", "",
		"`", "",
	)
	value = replacer.Replace(value)

	lines := strings.Split(value, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "- ")
		line = strings.TrimPrefix(line, "* ")
		lines[i] = line
	}

	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func sanitizeChatHistory(history []aiChatMessage) []llm.Message {
	if len(history) > maxChatHistoryItems {
		history = history[len(history)-maxChatHistoryItems:]
	}

	messages := make([]llm.Message, 0, len(history))
	for _, item := range history {
		role := strings.TrimSpace(item.Role)
		content := strings.TrimSpace(item.Content)
		if content == "" {
			continue
		}
		if role != "user" && role != "assistant" {
			continue
		}
		content = truncateRunes(content, maxChatMessageRunes)
		messages = append(messages, llm.Message{Role: role, Content: content})
	}

	return messages
}

func (s *Server) generateMeetingAnalysis(ctx context.Context, room string, now func() time.Time) (meetingAnalysis, error) {
	demo := demoMeetingAnalysis(room, now())
	contextText := analysisContext(demo)

	prompt := `Проанализируй встречу AlemLive и верни только валидный JSON без markdown.
Схема:
{
  "roomName": "string",
  "generatedAt": "RFC3339 string",
  "summary": [{"title": "string", "text": "string"}],
  "actionItems": [{"task": "string", "owner": "string", "due": "string", "priority": "High|Medium|Low"}],
  "transcript": [{"time": "00:00", "speaker": "string", "text": "string"}],
  "insights": {
    "sentiment": "string",
    "talkTime": [{"label": "string", "value": 0, "unit": "%"}],
    "speechRate": [{"label": "string", "value": 0, "unit": "wpm"}],
    "interruptions": [{"label": "string", "value": 0, "unit": "times"}],
    "engagement": [{"label": "string", "value": 0, "unit": "items"}]
  },
  "highlights": [{"time": "00:00", "title": "string", "text": "string", "type": "Decision|Risk|Action"}],
  "chapters": [{"start": "00:00", "end": "00:00", "title": "string", "text": "string"}]
}
Пиши на русском языке. Сохрани roomName и generatedAt из контекста.`

	answer, err := s.ai.Chat(ctx, []llm.Message{
		{Role: "system", Content: "Ты аналитик встреч. Возвращай только JSON по заданной схеме."},
		{Role: "user", Content: prompt + "\n\nКонтекст:\n" + contextText},
	}, llm.ChatOptions{
		Temperature: 0.2,
		MaxTokens:   1800,
	})
	if err != nil {
		return meetingAnalysis{}, err
	}

	var analysis meetingAnalysis
	if err := json.Unmarshal([]byte(extractJSONObject(answer)), &analysis); err != nil {
		return meetingAnalysis{}, err
	}
	if err := validateAnalysis(analysis); err != nil {
		return meetingAnalysis{}, err
	}
	if analysis.RoomName == "" {
		analysis.RoomName = room
	}
	if analysis.GeneratedAt == "" {
		analysis.GeneratedAt = now().UTC().Format(time.RFC3339)
	}

	return analysis, nil
}

func validateAnalysis(analysis meetingAnalysis) error {
	if len(analysis.Summary) == 0 || len(analysis.Transcript) == 0 {
		return errors.New("analysis is incomplete")
	}
	return nil
}

func extractJSONObject(raw string) string {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "```") {
		raw = strings.TrimPrefix(raw, "```json")
		raw = strings.TrimPrefix(raw, "```")
		raw = strings.TrimSuffix(raw, "```")
		raw = strings.TrimSpace(raw)
	}

	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start >= 0 && end > start {
		return raw[start : end+1]
	}

	return raw
}

func defaultReportContext(reportID string, now func() time.Time) string {
	if detail, ok := demoReportDetail(reportID, now()); ok {
		return reportDetailContext(detail)
	}

	analysis := demoMeetingAnalysis(firstNonEmpty(reportID, "alem-meeting"), now())
	return analysisContext(analysis)
}

func analysisContext(analysis meetingAnalysis) string {
	var b strings.Builder
	b.WriteString("Комната: ")
	b.WriteString(analysis.RoomName)
	b.WriteString("\nСоздано: ")
	b.WriteString(analysis.GeneratedAt)
	b.WriteString("\n\nSummary:\n")
	for _, item := range analysis.Summary {
		b.WriteString("- ")
		b.WriteString(item.Title)
		b.WriteString(": ")
		b.WriteString(item.Text)
		b.WriteString("\n")
	}
	b.WriteString("\nAction items:\n")
	for _, item := range analysis.ActionItems {
		b.WriteString("- ")
		b.WriteString(item.Task)
		b.WriteString(" / ")
		b.WriteString(item.Owner)
		b.WriteString(" / ")
		b.WriteString(item.Due)
		b.WriteString("\n")
	}
	b.WriteString("\nTranscript:\n")
	for _, line := range analysis.Transcript {
		b.WriteString(line.Time)
		b.WriteString(" ")
		b.WriteString(line.Speaker)
		b.WriteString(": ")
		b.WriteString(line.Text)
		b.WriteString("\n")
	}

	return b.String()
}

func truncateRunes(value string, limit int) string {
	if limit <= 0 || utf8.RuneCountInString(value) <= limit {
		return value
	}

	var b strings.Builder
	b.Grow(len(value))
	count := 0
	for _, r := range value {
		if count >= limit {
			break
		}
		b.WriteRune(r)
		count++
	}

	return b.String()
}
