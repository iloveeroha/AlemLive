package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/iloveeroha/AlemLive/backend/internal/llm"
)

const (
	maxRecordingUploadBytes = 500 << 20
	maxTranscriptRunes      = 60000
)

type meetingTranscriptionRequest struct {
	RoomName       string `json:"roomName"`
	TranscriptText string `json:"transcriptText"`
	Language       string `json:"language"`
	Participants   string `json:"participants"`
}

type meetingTranscriptionResponse struct {
	RoomName       string           `json:"roomName"`
	GeneratedAt    string           `json:"generatedAt"`
	TranscriptText string           `json:"transcriptText"`
	Transcript     []transcriptLine `json:"transcript"`
	Analysis       meetingAnalysis  `json:"analysis"`
	Model          string           `json:"model,omitempty"`
}

type recordingUploadInput struct {
	RoomName         string
	FileName         string
	ContentType      string
	Data             []byte
	Language         string
	Title            string
	Source           string
	Owner            string
	Folder           string
	Participants     string
	ParticipantNames string
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

	input, transcription, err := s.readTranscriptionInput(w, r)
	if err != nil {
		var apiErr *llm.APIError
		if errors.As(err, &apiErr) {
			writeError(w, http.StatusBadGateway, "STT service error: "+apiErr.Message)
			return
		}
		writeError(w, http.StatusBadGateway, "STT service error: "+err.Error())
		return
	}
	roomName := input.RoomName

	transcriptText := truncateRunes(strings.TrimSpace(transcription.Text), maxTranscriptRunes)
	if transcriptText == "" {
		writeError(w, http.StatusBadRequest, "transcript is empty")
		return
	}

	lines := transcriptLinesFromTranscription(transcription)
	if len(lines) == 0 {
		lines = transcriptLinesFromText(transcriptText)
	}
	lines = s.diarizeTranscript(r.Context(), input.FileName, input.ContentType, input.Data, input.ParticipantNames, lines)

	analysis, err := s.generateMeetingAnalysisFromTranscript(r.Context(), roomName, input.ParticipantNames, transcriptText, lines)
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

func (s *Server) readTranscriptionInput(w http.ResponseWriter, r *http.Request) (recordingUploadInput, llm.Transcription, error) {
	roomName := strings.TrimSpace(r.URL.Query().Get("roomName"))
	contentType := r.Header.Get("Content-Type")

	if strings.HasPrefix(contentType, "multipart/form-data") {
		input, err := s.readRecordingUploadInput(w, r, roomName)
		if err != nil {
			return recordingUploadInput{}, llm.Transcription{}, err
		}

		transcription, err := s.transcribeRecording(r.Context(), input)
		if err != nil {
			return recordingUploadInput{}, llm.Transcription{}, fmt.Errorf("speech-to-text service is unavailable: %w", err)
		}
		return input, transcription, nil
	}

	var req meetingTranscriptionRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 512*1024))
	if err := decoder.Decode(&req); err != nil {
		return recordingUploadInput{}, llm.Transcription{}, fmt.Errorf("invalid JSON body")
	}
	roomName = firstNonEmpty(roomName, req.RoomName, "alem-meeting")
	roomName, err := validateField("roomName", roomName)
	if err != nil {
		return recordingUploadInput{}, llm.Transcription{}, err
	}
	return recordingUploadInput{
		RoomName:         roomName,
		Language:         strings.TrimSpace(req.Language),
		ParticipantNames: strings.TrimSpace(req.Participants),
	}, llm.Transcription{Text: req.TranscriptText}, nil
}

func (s *Server) readRecordingUploadInput(w http.ResponseWriter, r *http.Request, defaultRoomName string) (recordingUploadInput, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRecordingUploadBytes)
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		return recordingUploadInput{}, fmt.Errorf("invalid multipart body or file is larger than 500MB")
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}

	roomName := firstNonEmpty(defaultRoomName, r.FormValue("roomName"), "alem-meeting")
	roomName, err := validateField("roomName", roomName)
	if err != nil {
		return recordingUploadInput{}, err
	}

	file, header, err := firstUploadedFile(r.MultipartForm)
	if err != nil {
		return recordingUploadInput{}, err
	}
	defer file.Close()

	data, err := readLimited(file, maxRecordingUploadBytes)
	if err != nil {
		return recordingUploadInput{}, err
	}

	return recordingUploadInput{
		RoomName:         roomName,
		FileName:         header.Filename,
		ContentType:      header.Header.Get("Content-Type"),
		Data:             data,
		Language:         strings.TrimSpace(firstNonEmpty(r.FormValue("language"), "ru")),
		Title:            strings.TrimSpace(r.FormValue("title")),
		Source:           strings.TrimSpace(r.FormValue("source")),
		Owner:            strings.TrimSpace(r.FormValue("owner")),
		Folder:           strings.TrimSpace(r.FormValue("folder")),
		Participants:     strings.TrimSpace(r.FormValue("participants")),
		ParticipantNames: strings.TrimSpace(r.FormValue("participantNames")),
	}, nil
}

func (s *Server) transcribeRecording(ctx context.Context, input recordingUploadInput) (llm.Transcription, error) {
	return s.stt.Transcribe(ctx, input.FileName, input.ContentType, input.Data, llm.TranscriptionOptions{
		Model:          s.cfg.STTModel,
		Language:       firstNonEmpty(input.Language, "ru"),
		ResponseFormat: "verbose_json",
	})
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

func (s *Server) generateMeetingAnalysisFromTranscript(ctx context.Context, roomName, participants, transcriptText string, lines []transcriptLine) (meetingAnalysis, error) {
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
  "insights": {
    "sentiment": "string",
    "talkTime": [{"label": "string", "value": 0, "unit": "%"}],
    "speechRate": [{"label": "string", "value": 0, "unit": "wpm"}],
    "interruptions": [{"label": "string", "value": 0, "unit": "times"}],
    "engagement": [{"label": "string", "value": 0, "unit": "items"}]
  },
  "highlights": [{"time": "00:00", "title": "string", "text": "string", "type": "Topic|Action|Question"}],
  "chapters": [{"start": "00:00", "end": "00:00", "title": "string", "text": "string", "points": ["string", "string"]}]
}
For each chapter, text is a short paragraph describing what was discussed, and points is 2-4 short bullet points with concrete topics or decisions from that chapter.
Use Russian for user-facing text. Preserve roomName. Do not return the full transcript; backend already has it.`

	contextText := transcriptAnalysisContext(roomName, participants, transcriptText, lines)
	answer, err := s.ai.Chat(ctx, []llm.Message{
		{Role: "system", Content: "You are a meeting analytics engine. Return only JSON that matches the requested schema. Do not copy the full transcript into the output. Use only known participant names or neutral Speaker/Speaker N labels for speakers; never invent speaker names from transcript words."},
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
	analysis.Transcript = normalizeTranscriptSpeakers(analysis.Transcript)
	analysis.Transcript = applyParticipantSpeakerNames(analysis.Transcript, participants)
	analysis.Transcript = annotateTranscriptSentiment(analysis.Transcript)
	if len(analysis.Keywords) == 0 {
		analysis.Keywords = extractKeywords(transcriptText, 8)
	}
	if strings.TrimSpace(analysis.Insights.Sentiment) == "" {
		analysis.Insights.Sentiment = "Информационная встреча"
	}
	if len(analysis.Insights.TalkTime) == 0 || hasOnlyGenericSpeakerStats(analysis.Insights.TalkTime) {
		analysis.Insights.TalkTime = speakerTalkTime(analysis.Transcript)
	}
	if len(analysis.Insights.SpeechRate) == 0 {
		analysis.Insights.SpeechRate = []metricValue{{Label: "Average", Value: estimatedSpeechRate(len(strings.Fields(transcriptText)), len(analysis.Transcript)), Unit: "wpm"}}
	}
	if len(analysis.Insights.Interruptions) == 0 {
		analysis.Insights.Interruptions = []metricValue{{Label: "Detected", Value: 0, Unit: "times"}}
	}
	if len(analysis.Insights.Engagement) == 0 {
		analysis.Insights.Engagement = []metricValue{{Label: "Transcript lines", Value: len(analysis.Transcript), Unit: "items"}}
	}
	if len(analysis.Highlights) == 0 {
		analysis.Highlights = fallbackHighlights(transcriptText)
	}
	if len(analysis.Chapters) == 0 {
		analysis.Chapters = fallbackChapters(analysis.Transcript)
	}
	if err := validateAnalysis(analysis); err != nil {
		return meetingAnalysis{}, err
	}
	return analysis, nil
}

func hasOnlyGenericSpeakerStats(values []metricValue) bool {
	if len(values) == 0 {
		return true
	}
	for _, value := range values {
		if strings.TrimSpace(value.Label) != "" && !strings.EqualFold(strings.TrimSpace(value.Label), "Speaker") {
			return false
		}
	}
	return true
}

func transcriptAnalysisContext(roomName, participants, transcriptText string, lines []transcriptLine) string {
	var b strings.Builder
	b.WriteString("Room: ")
	b.WriteString(roomName)
	if strings.TrimSpace(participants) != "" {
		b.WriteString("\nKnown participants: ")
		b.WriteString(strings.TrimSpace(participants))
	}
	b.WriteString("\nSpeaker policy: use Known participants for speaker names when possible. If unsure, use Speaker/Speaker N. Do not turn transcript words into speaker names.")
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
			Speaker: normalizeSpeakerLabel(segment.Speaker),
			Text:    text,
			Start:   segment.Start,
			End:     segment.End,
		})
		if i >= 199 {
			break
		}
	}
	return normalizeTranscriptSpeakers(lines)
}

func transcriptLinesFromText(text string) []transcriptLine {
	parts := splitTranscriptText(text)
	lines := make([]transcriptLine, 0, len(parts))
	start := 0.0
	for i, part := range parts {
		lines = append(lines, transcriptLine{
			Time:    formatTranscriptTime(float64(i * 30)),
			Speaker: "Speaker",
			Text:    part,
			Start:   start,
			End:     start + 30,
		})
		start += 30
		if i >= 199 {
			break
		}
	}
	return normalizeTranscriptSpeakers(lines)
}

var genericSpeakerPattern = regexp.MustCompile(`(?i)^speaker[_\s-]*(\d+)$`)

func isGenericSpeakerName(speaker string) bool {
	speaker = strings.TrimSpace(speaker)
	if speaker == "" || strings.EqualFold(speaker, "Speaker") {
		return true
	}
	return genericSpeakerPattern.MatchString(speaker)
}

func normalizeTranscriptSpeakers(lines []transcriptLine) []transcriptLine {
	out := make([]transcriptLine, len(lines))
	for i, line := range lines {
		line.Speaker = normalizeSpeakerLabel(line.Speaker)
		out[i] = line
	}
	return out
}

func normalizeSpeakerLabel(speaker string) string {
	speaker = strings.TrimSpace(speaker)
	if speaker == "" {
		return "Speaker"
	}

	if matches := genericSpeakerPattern.FindStringSubmatch(speaker); len(matches) == 2 {
		rawNumber := strings.TrimSpace(matches[1])
		if n, err := strconv.Atoi(rawNumber); err == nil {
			if strings.HasPrefix(rawNumber, "0") || len(rawNumber) > 1 {
				n++
			} else if n == 0 {
				n = 1
			}
			return fmt.Sprintf("Speaker %d", n)
		}
	}

	return speaker
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
	lines = normalizeTranscriptSpeakers(lines)
	lines = annotateTranscriptSentiment(lines)
	wordCount := len(strings.Fields(transcriptText))
	summary := fallbackSummarySections(transcriptText)
	actions := fallbackActionItems(transcriptText)
	highlights := fallbackHighlights(transcriptText)
	chapters := fallbackChapters(lines)
	talkTime := speakerTalkTime(lines)
	return meetingAnalysis{
		RoomName:    roomName,
		GeneratedAt: now.UTC().Format(time.RFC3339),
		Summary:     summary,
		ActionItems: actions,
		Transcript:  lines,
		Insights: meetingInsights{
			Sentiment: "Информационная встреча",
			TalkTime:  talkTime,
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
		Highlights: highlights,
		Chapters:   chapters,
		Keywords:   extractKeywords(transcriptText, 8),
	}
}

func fallbackSummarySections(transcriptText string) []summarySection {
	_ = transcriptText
	return []summarySection{
		{Title: "Транскрипт получен", Text: "Аудио распознано, но AI-анализ сейчас недоступен. Откройте вкладку транскрипта для исходного текста или повторите AI-запрос позже."},
	}
}

func fallbackActionItems(transcriptText string) []actionItem {
	_ = transcriptText
	return []actionItem{
		{Task: "Проверить транскрипт и подтвердить задачи вручную", Owner: "Team", Due: "После встречи", Priority: "Medium"},
	}
}

func fallbackHighlights(transcriptText string) []highlight {
	_ = transcriptText
	return []highlight{
		{Time: "00:00", Title: "Транскрипт готов", Text: "Speech-to-text завершился. Для финальных выводов нужен повторный AI-анализ.", Type: "Action"},
	}
}

func fallbackChapters(lines []transcriptLine) []chapter {
	end := formatTranscriptTime(float64(max(30, len(lines)*30)))
	return []chapter{
		{Start: "00:00", End: end, Title: "Разговор", Text: "Автоматически распознанная запись встречи."},
	}
}

func speakerTalkTime(lines []transcriptLine) []metricValue {
	counts := map[string]int{}
	order := make([]string, 0)
	total := 0
	for _, line := range lines {
		speaker := firstNonEmpty(strings.TrimSpace(line.Speaker), "Speaker")
		if _, ok := counts[speaker]; !ok {
			order = append(order, speaker)
		}
		counts[speaker]++
		total++
	}
	if total == 0 {
		return []metricValue{{Label: "Speaker", Value: 100, Unit: "%"}}
	}

	result := make([]metricValue, 0, len(counts))
	for _, speaker := range order {
		count := counts[speaker]
		if count == 0 {
			continue
		}
		result = append(result, metricValue{Label: speaker, Value: max(1, count*100/total), Unit: "%"})
	}
	return result
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
