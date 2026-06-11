package llm

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestChat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("unexpected authorization header: %s", got)
		}

		writeJSON(t, w, http.StatusOK, chatCompletionResponse{
			Choices: []struct {
				Message Message `json:"message"`
			}{
				{Message: Message{Role: "assistant", Content: "Hello"}},
			},
		})
	}))
	defer server.Close()

	client := New(server.URL, "test-key", "test-model", time.Second)
	answer, err := client.Chat(context.Background(), []Message{{Role: "user", Content: "Hi"}}, ChatOptions{})
	if err != nil {
		t.Fatalf("chat failed: %v", err)
	}
	if answer != "Hello" {
		t.Fatalf("unexpected answer: %q", answer)
	}
}

func TestChatRetriesTemporaryAPIError(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			writeJSON(t, w, http.StatusTooManyRequests, map[string]any{
				"error": map[string]string{"message": "rate limited"},
			})
			return
		}

		writeJSON(t, w, http.StatusOK, chatCompletionResponse{
			Choices: []struct {
				Message Message `json:"message"`
			}{
				{Message: Message{Role: "assistant", Content: "Recovered"}},
			},
		})
	}))
	defer server.Close()

	client := New(server.URL, "test-key", "test-model", time.Second)
	answer, err := client.Chat(context.Background(), []Message{{Role: "user", Content: "Hi"}}, ChatOptions{})
	if err != nil {
		t.Fatalf("chat failed: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
	if answer != "Recovered" {
		t.Fatalf("unexpected answer: %q", answer)
	}
}

func TestChatReturnsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, http.StatusUnauthorized, map[string]any{
			"error": map[string]string{"message": "bad key"},
		})
	}))
	defer server.Close()

	client := New(server.URL, "test-key", "test-model", time.Second)
	_, err := client.Chat(context.Background(), []Message{{Role: "user", Content: "Hi"}}, ChatOptions{})
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != http.StatusUnauthorized || apiErr.Message != "bad key" {
		t.Fatalf("unexpected api error: %#v", apiErr)
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, status int, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}
