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

func TestWorkspaceUtilityEndpoints(t *testing.T) {
	handler := NewServer(config.Config{TokenTTL: time.Hour})

	for _, path := range []string{"/api/notifications", "/api/profile", "/api/locales"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		response := httptest.NewRecorder()

		handler.ServeHTTP(response, request)

		if response.Code != http.StatusOK {
			t.Fatalf("%s expected status 200, got %d: %s", path, response.Code, response.Body.String())
		}
	}
}

func TestAuthConfigEndpoint(t *testing.T) {
	handler := NewServer(config.Config{
		TokenTTL:          time.Hour,
		KeycloakIssuerURL: "https://keycloak.example/realms/alem",
		KeycloakClientID:  "alemlive",
	})

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/auth/config", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("expected auth config status 200, got %d: %s", response.Code, response.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode auth config: %v", err)
	}
	if payload["enabled"] != true || payload["clientId"] != "alemlive" {
		t.Fatalf("unexpected auth config: %#v", payload)
	}
}

func TestKeycloakEnabledRequiresBearerToken(t *testing.T) {
	handler := NewServer(config.Config{
		TokenTTL:          time.Hour,
		KeycloakIssuerURL: "https://keycloak.example/realms/alem",
		KeycloakClientID:  "alemlive",
	})

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/profile", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", response.Code, response.Body.String())
	}
}

func TestRoomActions(t *testing.T) {
	handler := NewServer(config.Config{TokenTTL: time.Hour})

	linkResponse := httptest.NewRecorder()
	handler.ServeHTTP(linkResponse, httptest.NewRequest(http.MethodGet, "/api/rooms/alem-meeting/link", nil))
	if linkResponse.Code != http.StatusOK {
		t.Fatalf("expected link status 200, got %d: %s", linkResponse.Code, linkResponse.Body.String())
	}

	var linkPayload map[string]string
	if err := json.Unmarshal(linkResponse.Body.Bytes(), &linkPayload); err != nil {
		t.Fatalf("decode link response: %v", err)
	}
	if linkPayload["joinUrl"] == "" {
		t.Fatalf("expected join url: %#v", linkPayload)
	}

	settingsResponse := httptest.NewRecorder()
	handler.ServeHTTP(settingsResponse, httptest.NewRequest(http.MethodGet, "/api/rooms/alem-meeting/settings", nil))
	if settingsResponse.Code != http.StatusOK {
		t.Fatalf("expected settings status 200, got %d: %s", settingsResponse.Code, settingsResponse.Body.String())
	}

	leaveResponse := httptest.NewRecorder()
	handler.ServeHTTP(leaveResponse, httptest.NewRequest(http.MethodPost, "/api/rooms/alem-meeting/leave", nil))
	if leaveResponse.Code != http.StatusOK {
		t.Fatalf("expected leave status 200, got %d: %s", leaveResponse.Code, leaveResponse.Body.String())
	}
}

func TestDevicePreferenceAndMeetingEvent(t *testing.T) {
	handler := NewServer(config.Config{TokenTTL: time.Hour})

	deviceBody := bytes.NewBufferString(`{"roomName":"alem-meeting","userName":"Madi","device":"mic","enabled":false}`)
	deviceResponse := httptest.NewRecorder()
	handler.ServeHTTP(deviceResponse, httptest.NewRequest(http.MethodPost, "/api/devices", deviceBody))
	if deviceResponse.Code != http.StatusOK {
		t.Fatalf("expected device status 200, got %d: %s", deviceResponse.Code, deviceResponse.Body.String())
	}

	eventBody := bytes.NewBufferString(`{"roomName":"alem-meeting","userName":"Madi","event":"copy_room"}`)
	eventResponse := httptest.NewRecorder()
	handler.ServeHTTP(eventResponse, httptest.NewRequest(http.MethodPost, "/api/meetings/events", eventBody))
	if eventResponse.Code != http.StatusOK {
		t.Fatalf("expected event status 200, got %d: %s", eventResponse.Code, eventResponse.Body.String())
	}
}
