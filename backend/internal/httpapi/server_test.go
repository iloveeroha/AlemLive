package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/iloveeroha/AlemLive/backend/internal/config"
)

func TestCreateLiveKitToken(t *testing.T) {
	handler := NewServer(config.Config{
		LiveKitURL:    "wss://alem-livekit.example",
		LiveKitAPIKey: "key",
		LiveKitSecret: "secret",
		TokenTTL:      time.Hour,
	})

	body := bytes.NewBufferString(`{"roomName":"alem-meeting","userName":"Madi"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/livekit/token", body)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}

	var payload map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if payload["serverUrl"] != "wss://alem-livekit.example" {
		t.Fatalf("unexpected serverUrl: %#v", payload)
	}
	if payload["token"] == "" {
		t.Fatal("token should not be empty")
	}
}

func TestCreateLiveKitTokenRequiresConfig(t *testing.T) {
	handler := NewServer(config.Config{TokenTTL: time.Hour})
	request := httptest.NewRequest(http.MethodPost, "/api/livekit/token", bytes.NewBufferString(`{}`))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", response.Code)
	}
}

func TestMeetingAnalysis(t *testing.T) {
	handler := NewServer(config.Config{TokenTTL: time.Hour})
	request := httptest.NewRequest(http.MethodGet, "/api/meetings/analysis?roomName=alem-meeting", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}

	var payload meetingAnalysis
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if payload.RoomName != "alem-meeting" {
		t.Fatalf("unexpected room name: %s", payload.RoomName)
	}
	if len(payload.Summary) == 0 || len(payload.ActionItems) == 0 || len(payload.Transcript) == 0 {
		t.Fatalf("analysis payload is incomplete: %#v", payload)
	}
	if len(payload.Chapters) == 0 || len(payload.Highlights) == 0 {
		t.Fatalf("timeline payload is incomplete: %#v", payload)
	}
}

func TestAIChatRequiresConfig(t *testing.T) {
	handler := NewServer(config.Config{TokenTTL: time.Hour})
	request := httptest.NewRequest(http.MethodPost, "/api/ai/chat", bytes.NewBufferString(`{"message":"hello"}`))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", response.Code)
	}
}

func TestAIStatus(t *testing.T) {
	handler := NewServer(config.Config{
		TokenTTL:   time.Hour,
		LLMBaseURL: "https://llm.example",
		LLMAPIKey:  "test-key",
		LLMModel:   "test-model",
		LLMTimeout: time.Second,
	})
	request := httptest.NewRequest(http.MethodGet, "/api/ai/status", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}

	var payload aiStatusResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !payload.Configured {
		t.Fatal("expected configured AI status")
	}
	if payload.BaseURL != "https://llm.example" || payload.Model != "test-model" {
		t.Fatalf("unexpected status payload: %#v", payload)
	}
}

func TestAIChat(t *testing.T) {
	llmServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("unexpected authorization header: %s", got)
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"role": "assistant", "content": "AI ответ готов"}},
			},
		})
	}))
	defer llmServer.Close()

	handler := NewServer(config.Config{
		TokenTTL:   time.Hour,
		LLMBaseURL: llmServer.URL,
		LLMAPIKey:  "test-key",
		LLMModel:   "alemgpt-intent",
		LLMTimeout: time.Second,
	})
	request := httptest.NewRequest(http.MethodPost, "/api/ai/chat", bytes.NewBufferString(`{"message":"Какие задачи?"}`))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}

	var payload aiChatResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Answer != "AI ответ готов" {
		t.Fatalf("unexpected answer: %q", payload.Answer)
	}
}

func TestAIChatTrimsUnicodeHistorySafely(t *testing.T) {
	llmServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode llm request: %v", err)
		}

		for _, message := range payload.Messages {
			if !utf8.ValidString(message.Content) {
				t.Fatalf("message is not valid utf-8: %q", message.Content)
			}
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"role": "assistant", "content": "ok"}},
			},
		})
	}))
	defer llmServer.Close()

	handler := NewServer(config.Config{
		TokenTTL:   time.Hour,
		LLMBaseURL: llmServer.URL,
		LLMAPIKey:  "test-key",
		LLMModel:   "alemgpt-intent",
		LLMTimeout: time.Second,
	})

	longUnicode := strings.Repeat("қ", maxChatMessageRunes+10)
	body := bytes.NewBufferString(`{
		"message":"hello",
		"history":[{"role":"user","content":"` + longUnicode + `"}]
	}`)
	request := httptest.NewRequest(http.MethodPost, "/api/ai/chat", body)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
}
