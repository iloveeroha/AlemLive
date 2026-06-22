package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"
	"time"
)

var ErrNotConfigured = errors.New("llm client is not configured")

const (
	defaultTemperature = 0.3
	defaultMaxTokens   = 700
	maxErrorBodyBytes  = 4096
	maxResponseBytes   = 2 << 20
	maxAttempts        = 3
)

type Client struct {
	BaseURL    string
	APIKey     string
	Model      string
	HTTPClient *http.Client
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatOptions struct {
	Temperature float64
	MaxTokens   int
}

type TranscriptionOptions struct {
	Model          string
	Language       string
	Prompt         string
	ResponseFormat string
	Temperature    float64
}

type Transcription struct {
	Text     string                 `json:"text"`
	Segments []TranscriptionSegment `json:"segments,omitempty"`
}

type TranscriptionSegment struct {
	ID      int     `json:"id,omitempty"`
	Start   float64 `json:"start"`
	End     float64 `json:"end"`
	Text    string  `json:"text"`
	Speaker string  `json:"speaker,omitempty"`
}

type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("llm status %d", e.StatusCode)
	}
	return fmt.Sprintf("llm status %d: %s", e.StatusCode, e.Message)
}

type chatCompletionRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

type transcriptionResponse struct {
	Text     string `json:"text"`
	Segments []struct {
		ID           int     `json:"id,omitempty"`
		Start        float64 `json:"start"`
		End          float64 `json:"end"`
		Text         string  `json:"text"`
		Speaker      string  `json:"speaker"`
		SpeakerLabel string  `json:"speaker_label"`
		Label        string  `json:"label"`
	} `json:"segments,omitempty"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

func New(baseURL, apiKey, model string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 60 * time.Second
	}

	return &Client{
		BaseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		APIKey:  strings.TrimSpace(apiKey),
		Model:   strings.TrimSpace(model),
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) Configured() bool {
	return c != nil && c.BaseURL != "" && c.APIKey != "" && c.Model != ""
}

func (c *Client) Chat(ctx context.Context, messages []Message, opts ChatOptions) (string, error) {
	if !c.Configured() {
		return "", ErrNotConfigured
	}
	if len(messages) == 0 {
		return "", errors.New("messages are required")
	}
	opts = normalizeOptions(opts)

	payload := chatCompletionRequest{
		Model:       c.Model,
		Messages:    messages,
		Temperature: opts.Temperature,
		MaxTokens:   opts.MaxTokens,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode request: %w", err)
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		answer, err := c.doChat(ctx, body)
		if err == nil {
			return answer, nil
		}
		lastErr = err

		if !shouldRetry(err) || attempt == maxAttempts {
			break
		}
		if err := sleepBeforeRetry(ctx, attempt); err != nil {
			return "", err
		}
	}

	return "", lastErr
}

func (c *Client) Transcribe(ctx context.Context, fileName, contentType string, data []byte, opts TranscriptionOptions) (Transcription, error) {
	if !c.Configured() {
		return Transcription{}, ErrNotConfigured
	}
	if len(data) == 0 {
		return Transcription{}, errors.New("audio data is required")
	}
	if strings.TrimSpace(fileName) == "" {
		fileName = "recording.webm"
	}
	if strings.TrimSpace(contentType) == "" {
		contentType = "application/octet-stream"
	}

	model := strings.TrimSpace(opts.Model)
	if model == "" {
		model = c.Model
	}
	responseFormat := strings.TrimSpace(opts.ResponseFormat)
	if responseFormat == "" {
		responseFormat = "verbose_json"
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, escapeMultipartFileName(fileName)))
	header.Set("Content-Type", contentType)
	part, err := writer.CreatePart(header)
	if err != nil {
		return Transcription{}, fmt.Errorf("create multipart file: %w", err)
	}
	if _, err := part.Write(data); err != nil {
		return Transcription{}, fmt.Errorf("write multipart file: %w", err)
	}
	if err := writer.WriteField("model", model); err != nil {
		return Transcription{}, fmt.Errorf("write model field: %w", err)
	}
	if opts.Language != "" {
		if err := writer.WriteField("language", opts.Language); err != nil {
			return Transcription{}, fmt.Errorf("write language field: %w", err)
		}
	}
	if opts.Prompt != "" {
		if err := writer.WriteField("prompt", opts.Prompt); err != nil {
			return Transcription{}, fmt.Errorf("write prompt field: %w", err)
		}
	}
	if err := writer.WriteField("response_format", responseFormat); err != nil {
		return Transcription{}, fmt.Errorf("write response_format field: %w", err)
	}
	if opts.Temperature > 0 {
		if err := writer.WriteField("temperature", fmt.Sprintf("%.2f", opts.Temperature)); err != nil {
			return Transcription{}, fmt.Errorf("write temperature field: %w", err)
		}
	}
	if err := writer.Close(); err != nil {
		return Transcription{}, fmt.Errorf("close multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/v1/audio/transcriptions", &body)
	if err != nil {
		return Transcription{}, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return Transcription{}, fmt.Errorf("call transcription: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes*16))
	if err != nil {
		return Transcription{}, fmt.Errorf("read response: %w", err)
	}

	var parsed transcriptionResponse
	if len(raw) > 0 && strings.Contains(resp.Header.Get("Content-Type"), "application/json") {
		if err := json.Unmarshal(raw, &parsed); err != nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return Transcription{}, fmt.Errorf("decode response: %w", err)
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Transcription{}, apiErrorFromTranscriptionResponse(resp.StatusCode, parsed, raw)
	}

	text := strings.TrimSpace(parsed.Text)
	if text == "" {
		text = strings.TrimSpace(string(raw))
	}
	if text == "" {
		return Transcription{}, errors.New("transcription response is empty")
	}

	result := Transcription{Text: text}
	for _, segment := range parsed.Segments {
		if strings.TrimSpace(segment.Text) == "" {
			continue
		}
		result.Segments = append(result.Segments, TranscriptionSegment{
			ID:      segment.ID,
			Start:   segment.Start,
			End:     segment.End,
			Text:    strings.TrimSpace(segment.Text),
			Speaker: firstNonEmpty(segment.Speaker, segment.SpeakerLabel, segment.Label),
		})
	}

	return result, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func escapeMultipartFileName(fileName string) string {
	return strings.NewReplacer("\\", "\\\\", `"`, "\\\"").Replace(fileName)
}

func (c *Client) doChat(ctx context.Context, body []byte) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("call llm: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	var parsed chatCompletionResponse
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &parsed); err != nil {
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				return "", apiErrorFromResponse(resp.StatusCode, parsed, raw)
			}
			return "", fmt.Errorf("decode response: %w", err)
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", apiErrorFromResponse(resp.StatusCode, parsed, raw)
	}

	if len(parsed.Choices) == 0 {
		return "", errors.New("llm response has no choices")
	}

	answer := strings.TrimSpace(parsed.Choices[0].Message.Content)
	if answer == "" {
		return "", errors.New("llm response is empty")
	}

	return answer, nil
}

func apiErrorFromTranscriptionResponse(status int, parsed transcriptionResponse, raw []byte) error {
	if parsed.Error != nil && strings.TrimSpace(parsed.Error.Message) != "" {
		return &APIError{StatusCode: status, Message: strings.TrimSpace(parsed.Error.Message)}
	}
	message := strings.TrimSpace(string(raw))
	if message == "" {
		message = http.StatusText(status)
	}
	return &APIError{StatusCode: status, Message: message}
}

func normalizeOptions(opts ChatOptions) ChatOptions {
	if opts.Temperature == 0 {
		opts.Temperature = defaultTemperature
	}
	if opts.MaxTokens <= 0 {
		opts.MaxTokens = defaultMaxTokens
	}
	return opts
}

func apiErrorFromResponse(status int, parsed chatCompletionResponse, raw []byte) error {
	if parsed.Error != nil && strings.TrimSpace(parsed.Error.Message) != "" {
		return &APIError{StatusCode: status, Message: strings.TrimSpace(parsed.Error.Message)}
	}
	message := strings.TrimSpace(string(raw))
	if message == "" {
		message = http.StatusText(status)
	}
	return &APIError{StatusCode: status, Message: message}
}

func shouldRetry(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusTooManyRequests || apiErr.StatusCode >= http.StatusInternalServerError
	}
	return false
}

func sleepBeforeRetry(ctx context.Context, attempt int) error {
	delay := time.Duration(attempt*attempt) * 250 * time.Millisecond
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
