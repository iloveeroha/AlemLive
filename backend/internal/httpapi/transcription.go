package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/iloveeroha/AlemLive/backend/internal/llm"
)

const (
	maxRecordingUploadBytes = 50 << 20
	maxTranscriptRunes      = 60000
)

type meetingTranscriptionRequest struct {
	RoomName       string `json:"roomName"`
	TranscriptText string `json:"transcriptText"`
	Language       string `json:"language"`
}

type meetingTranscriptionResponse struct {
	RoomName       string           `json:"roomName"`
	GeneratedAt    string           `json:"generatedAt"`
	TranscriptText string           `json:"transcriptText"`
	Transcript     []transcriptLine `json:"transcript"`
	Analysis       meetingAnalysis  `json:"analysis"`
	Model          string           `json:"model,omitempty"`
}

func (s *Server) meetingTranscription(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}

	if s.stt == nil || !s.stt.Configured() {
		writeError(w, http.StatusServiceUnavailable, "Speech-to-text backend is not configured")
		return
	}

	roomName, transcription, err := s.readTranscriptionInput(w, r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	transcriptText := truncateRunes(strings.TrimSpace(transcription.Text), maxTranscriptRunes)
	if transcriptText == "" {
		writeError(w, http.StatusBadRequest, "transcript is empty")
		return
	}

	lines := transcriptLinesFromTranscription(transcription)
	if len(lines) == 0 {
		lines = transcriptLinesFromText(transcriptText)
	}

	analysis, err := s.generateMeetingAnalysisFromTranscript(r.Context(), roomName, transcriptText, lines)
	if err != nil || s.ai == nil || !s.ai.Configured() {
		analysis = fallbackAnalysisFromTranscript(roomName, transcriptText, lines, s.clock())
	}

	writeJSON(w, http.StatusOK, meetingTranscriptionResponse{
		RoomName:       roomName,
		GeneratedAt:    s.clock().UTC().Format(time.RFC3339),
		TranscriptText: transcriptText,
		Transcript:     lines,
		Analysis:       analysis,
		Model:          s.cfg.STTModel,
	})
}

func (s *Server) readTranscriptionInput(w http.ResponseWriter, r *http.Request) (string, llm.Transcription, error) {
	roomName := strings.TrimSpace(r.URL.Query().Get("roomName"))
	contentType := r.Header.Get("Content-Type")

	if strings.HasPrefix(contentType, "multipart/form-data") {
		r.Body = http.MaxBytesReader(w, r.Body, maxRecordingUploadBytes)
		if err := r.ParseMultipartForm(8 << 20); err != nil {
			return "", llm.Transcription{}, fmt.Errorf("invalid multipart body")
		}
		roomName = firstNonEmpty(roomName, r.FormValue("roomName"), "alem-meeting")
		roomName, err := validateField("roomName", roomName)
		if err != nil {
			return "", llm.Transcription{}, err
		}

		file, header, err := firstUploadedFile(r.MultipartForm)
		if err != nil {
			return "", llm.Transcription{}, err
		}
		defer file.Close()

		data, err := readLimited(file, maxRecordingUploadBytes)
		if err != nil {
			return "", llm.Transcription{}, err
		}

		language := strings.TrimSpace(firstNonEmpty(r.FormValue("language"), "ru"))
		transcription, err := s.stt.Transcribe(r.Context(), header.Filename, header.Header.Get("Content-Type"), data, llm.TranscriptionOptions{
			Model:          s.cfg.STTModel,
			Language:       language,
			ResponseFormat: "verbose_json",
		})
		if err != nil {
			return "", llm.Transcription{}, fmt.Errorf("speech-to-text service is unavailable")
		}
		return roomName, transcription, nil
	}

	var req meetingTranscriptionRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 512*1024))
	if err := decoder.Decode(&req); err != nil {
		return "", llm.Transcription{}, fmt.Errorf("invalid JSON body")
	}
	roomName = firstNonEmpty(roomName, req.RoomName, "alem-meeting")
	roomName, err := validateField("roomName", roomName)
	if err != nil {
		return "", llm.Transcription{}, err
	}
	return roomName, llm.Transcription{Text: req.TranscriptText}, nil
}

func firstUploadedFile(form *multipart.Form) (multipart.File, *multipart.FileHeader, error) {
	if form == nil {
		return nil, nil, fmt.Errorf("file is required")
	}
	for _, field := range []string{"file", "audio", "recording"} {
		files := form.File[field]
		if len(files) == 0 {
			continue
		}
		file, err := files[0].Open()
		if err != nil {
			return nil, nil, fmt.Errorf("could not open uploaded file")
		}
		return file, files[0], nil
	}
	return nil, nil, fmt.Errorf("file is required")
}

func readLimited(reader io.Reader, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, fmt.Errorf("could not read uploaded file")
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("uploaded file is too large")
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("uploaded file is empty")
	}
	return data, nil
}

func (s *Server) generateMeetingAnalysisFromTranscript(ctx context.Context, roomName, transcriptText string, lines []transcriptLine) (meetingAnalysis, error) {
	if s.ai == nil || !s.ai.Configured() {
		return meetingAnalysis{}, llm.ErrNotConfigured
	}

	prompt := `Analyze this meeting transcript and return only valid JSON without markdown.
Schema:
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
Use Russian for user-facing text. Preserve roomName.`

	contextText := transcriptAnalysisContext(roomName, transcriptText, lines)
	answer, err := s.ai.Chat(ctx, []llm.Message{
		{Role: "system", Content: "You are a meeting analytics engine. Return only JSON that matches the requested schema."},
		{Role: "user", Content: prompt + "\n\n" + contextText},
	}, llm.ChatOptions{
		Temperature: 0.2,
		MaxTokens:   2200,
	})
	if err != nil {
		return meetingAnalysis{}, err
	}

	var analysis meetingAnalysis
	if err := json.Unmarshal([]byte(extractJSONObject(answer)), &analysis); err != nil {
		return meetingAnalysis{}, err
	}
	if analysis.RoomName == "" {
		analysis.RoomName = roomName
	}
	if analysis.GeneratedAt == "" {
		analysis.GeneratedAt = s.clock().UTC().Format(time.RFC3339)
	}
	if len(analysis.Transcript) == 0 {
		analysis.Transcript = lines
	}
	if err := validateAnalysis(analysis); err != nil {
		return meetingAnalysis{}, err
	}
	return analysis, nil
}

func transcriptAnalysisContext(roomName, transcriptText string, lines []transcriptLine) string {
	var b strings.Builder
	b.WriteString("Room: ")
	b.WriteString(roomName)
	b.WriteString("\n\nTranscript lines:\n")
	for _, line := range lines {
		b.WriteString(line.Time)
		b.WriteString(" ")
		b.WriteString(line.Speaker)
		b.WriteString(": ")
		b.WriteString(line.Text)
		b.WriteString("\n")
	}
	if len(lines) == 0 {
		b.WriteString(transcriptText)
	}
	return truncateRunes(b.String(), maxChatContextRunes)
}

func transcriptLinesFromTranscription(transcription llm.Transcription) []transcriptLine {
	lines := make([]transcriptLine, 0, len(transcription.Segments))
	for i, segment := range transcription.Segments {
		text := strings.TrimSpace(segment.Text)
		if text == "" {
			continue
		}
		lines = append(lines, transcriptLine{
			Time:    formatTranscriptTime(segment.Start),
			Speaker: "Speaker",
			Text:    text,
		})
		if i >= 199 {
			break
		}
	}
	return lines
}

func transcriptLinesFromText(text string) []transcriptLine {
	parts := splitTranscriptText(text)
	lines := make([]transcriptLine, 0, len(parts))
	for i, part := range parts {
		lines = append(lines, transcriptLine{
			Time:    formatTranscriptTime(float64(i * 30)),
			Speaker: "Speaker",
			Text:    part,
		})
		if i >= 199 {
			break
		}
	}
	return lines
}

func splitTranscriptText(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	rawParts := strings.FieldsFunc(text, func(r rune) bool {
		return r == '\n' || r == '.' || r == '!' || r == '?'
	})
	parts := make([]string, 0, len(rawParts))
	for _, part := range rawParts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if utf8.RuneCountInString(part) > 260 {
			part = truncateRunes(part, 260)
		}
		parts = append(parts, part)
	}
	if len(parts) == 0 {
		return []string{text}
	}
	return parts
}

func fallbackAnalysisFromTranscript(roomName, transcriptText string, lines []transcriptLine, now time.Time) meetingAnalysis {
	if len(lines) == 0 {
		lines = transcriptLinesFromText(transcriptText)
	}
	firstText := truncateRunes(strings.TrimSpace(transcriptText), 600)
	wordCount := len(strings.Fields(transcriptText))
	return meetingAnalysis{
		RoomName:    roomName,
		GeneratedAt: now.UTC().Format(time.RFC3339),
		Summary: []summarySection{
			{Title: "Transcript captured", Text: firstNonEmpty(firstText, "Audio was transcribed successfully.")},
		},
		ActionItems: []actionItem{
			{Task: "Review generated transcript and confirm action items", Owner: "Team", Due: "After meeting", Priority: "Medium"},
		},
		Transcript: lines,
		Insights: meetingInsights{
			Sentiment: "Needs review",
			TalkTime: []metricValue{
				{Label: "Speaker", Value: 100, Unit: "%"},
			},
			SpeechRate: []metricValue{
				{Label: "Average", Value: estimatedSpeechRate(wordCount, len(lines)), Unit: "wpm"},
			},
			Interruptions: []metricValue{
				{Label: "Detected", Value: 0, Unit: "times"},
			},
			Engagement: []metricValue{
				{Label: "Transcript lines", Value: len(lines), Unit: "items"},
				{Label: "Words", Value: wordCount, Unit: "items"},
			},
		},
		Highlights: []highlight{
			{Time: "00:00", Title: "Transcript ready", Text: "Speech-to-text completed. AI review is recommended for final highlights.", Type: "Action"},
		},
		Chapters: []chapter{
			{Start: "00:00", End: formatTranscriptTime(float64(max(30, len(lines)*30))), Title: "Conversation", Text: "Automatically transcribed meeting conversation."},
		},
	}
}

func estimatedSpeechRate(wordCount, lineCount int) int {
	if wordCount <= 0 {
		return 0
	}
	minutes := max(1, lineCount/2)
	return wordCount / minutes
}

func formatTranscriptTime(seconds float64) string {
	if seconds < 0 {
		seconds = 0
	}
	total := int(seconds + 0.5)
	hours := total / 3600
	minutes := (total % 3600) / 60
	secs := total % 60
	if hours > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, secs)
	}
	return fmt.Sprintf("%02d:%02d", minutes, secs)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
