package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var ErrNotConfigured = errors.New("llm client is not configured")

const (
	defaultTemperature = 0.3
	defaultMaxTokens   = 700
	maxErrorBodyBytes  = 4096
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

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes))
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
