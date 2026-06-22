package diarization

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
	"strconv"
	"strings"
	"time"
)

var ErrNotConfigured = errors.New("diarization client is not configured")

const (
	defaultTimeout    = 15 * time.Minute
	maxResponseBytes  = 2 << 20
	maxErrorBodyBytes = 4096
)

type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

type Options struct {
	Participants string
	MinSpeakers  int
	MaxSpeakers  int
}

type Result struct {
	Segments []Segment `json:"segments"`
}

type Segment struct {
	Start   float64 `json:"start"`
	End     float64 `json:"end"`
	Speaker string  `json:"speaker"`
}

type responsePayload struct {
	Segments []segmentPayload `json:"segments"`
	Error    *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type segmentPayload struct {
	Start        float64 `json:"start"`
	End          float64 `json:"end"`
	Speaker      string  `json:"speaker"`
	SpeakerLabel string  `json:"speaker_label"`
	Label        string  `json:"label"`
}

type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("diarization status %d", e.StatusCode)
	}
	return fmt.Sprintf("diarization status %d: %s", e.StatusCode, e.Message)
}

func New(baseURL, apiKey string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	return &Client{
		BaseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		APIKey:  strings.TrimSpace(apiKey),
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) Configured() bool {
	return c != nil && c.BaseURL != ""
}

func (c *Client) Diarize(ctx context.Context, fileName, contentType string, data []byte, opts Options) (Result, error) {
	if !c.Configured() {
		return Result{}, ErrNotConfigured
	}
	if len(data) == 0 {
		return Result{}, errors.New("audio data is required")
	}
	if strings.TrimSpace(fileName) == "" {
		fileName = "recording.wav"
	}
	if strings.TrimSpace(contentType) == "" {
		contentType = "application/octet-stream"
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, escapeMultipartFileName(fileName)))
	header.Set("Content-Type", contentType)
	part, err := writer.CreatePart(header)
	if err != nil {
		return Result{}, fmt.Errorf("create multipart file: %w", err)
	}
	if _, err := part.Write(data); err != nil {
		return Result{}, fmt.Errorf("write multipart file: %w", err)
	}
	if strings.TrimSpace(opts.Participants) != "" {
		if err := writer.WriteField("participants", strings.TrimSpace(opts.Participants)); err != nil {
			return Result{}, fmt.Errorf("write participants field: %w", err)
		}
	}
	if opts.MinSpeakers > 0 {
		if err := writer.WriteField("min_speakers", strconv.Itoa(opts.MinSpeakers)); err != nil {
			return Result{}, fmt.Errorf("write min_speakers field: %w", err)
		}
	}
	if opts.MaxSpeakers > 0 {
		if err := writer.WriteField("max_speakers", strconv.Itoa(opts.MaxSpeakers)); err != nil {
			return Result{}, fmt.Errorf("write max_speakers field: %w", err)
		}
	}
	if err := writer.Close(); err != nil {
		return Result{}, fmt.Errorf("close multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/diarize", &body)
	if err != nil {
		return Result{}, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return Result{}, fmt.Errorf("call diarization: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return Result{}, fmt.Errorf("read response: %w", err)
	}

	var parsed responsePayload
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &parsed); err != nil {
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				return Result{}, apiErrorFromResponse(resp.StatusCode, parsed, raw)
			}
			return Result{}, fmt.Errorf("decode response: %w", err)
		}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Result{}, apiErrorFromResponse(resp.StatusCode, parsed, raw)
	}

	result := Result{Segments: make([]Segment, 0, len(parsed.Segments))}
	for _, segment := range parsed.Segments {
		speaker := firstNonEmpty(segment.Speaker, segment.SpeakerLabel, segment.Label)
		if speaker == "" || segment.End <= segment.Start {
			continue
		}
		result.Segments = append(result.Segments, Segment{
			Start:   segment.Start,
			End:     segment.End,
			Speaker: speaker,
		})
	}
	if len(result.Segments) == 0 {
		return Result{}, errors.New("diarization response has no segments")
	}
	return result, nil
}

func apiErrorFromResponse(status int, parsed responsePayload, raw []byte) error {
	if parsed.Error != nil && strings.TrimSpace(parsed.Error.Message) != "" {
		return &APIError{StatusCode: status, Message: strings.TrimSpace(parsed.Error.Message)}
	}
	message := strings.TrimSpace(string(raw))
	if message == "" {
		message = http.StatusText(status)
	}
	if len(message) > maxErrorBodyBytes {
		message = message[:maxErrorBodyBytes]
	}
	return &APIError{StatusCode: status, Message: message}
}

func escapeMultipartFileName(fileName string) string {
	return strings.NewReplacer("\\", "\\\\", `"`, "\\\"").Replace(fileName)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
