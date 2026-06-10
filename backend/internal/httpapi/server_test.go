package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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
