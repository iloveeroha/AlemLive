package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/iloveeroha/AlemLive/backend/internal/config"
	livekitservice "github.com/iloveeroha/AlemLive/backend/internal/livekit"
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

func TestMobileRoomFlowEndpoints(t *testing.T) {
	handler := NewServer(config.Config{TokenTTL: time.Hour})
	server := httptest.NewServer(handler)
	defer server.Close()

	createBody := bytes.NewBufferString(`{"roomName":"Mobile Sync","initialMicEnabled":true,"initialCameraEnabled":false}`)
	createResponse, err := http.Post(server.URL+"/api/rooms/create", "application/json", createBody)
	if err != nil {
		t.Fatalf("create room request: %v", err)
	}
	defer createResponse.Body.Close()
	if createResponse.StatusCode != http.StatusOK {
		t.Fatalf("expected create status 200, got %d", createResponse.StatusCode)
	}

	var created struct {
		RoomID       string           `json:"roomId"`
		RoomName     string           `json:"roomName"`
		OwnerID      string           `json:"ownerId"`
		IsOwner      bool             `json:"isOwner"`
		Participants []map[string]any `json:"participants"`
	}
	if err := json.NewDecoder(createResponse.Body).Decode(&created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.RoomID == "" || created.RoomName != "Mobile Sync" || created.OwnerID == "" || !created.IsOwner {
		t.Fatalf("unexpected create payload: %#v", created)
	}
	if len(created.Participants) != 1 {
		t.Fatalf("expected current participant in create payload: %#v", created.Participants)
	}

	eventsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/rooms/" + created.RoomID + "/events"
	conn, _, err := websocket.DefaultDialer.Dial(eventsURL, nil)
	if err != nil {
		t.Fatalf("connect room events: %v", err)
	}
	defer conn.Close()

	var event roomEventEnvelope
	if err := conn.ReadJSON(&event); err != nil {
		t.Fatalf("read initial room event: %v", err)
	}
	if event.Type != "owner_changed" || event.Payload["ownerId"] != created.OwnerID {
		t.Fatalf("unexpected initial event: %#v", event)
	}

	participantsResponse, err := http.Get(server.URL + "/api/rooms/" + created.RoomID + "/participants")
	if err != nil {
		t.Fatalf("participants request: %v", err)
	}
	defer participantsResponse.Body.Close()
	if participantsResponse.StatusCode != http.StatusOK {
		t.Fatalf("expected participants status 200, got %d", participantsResponse.StatusCode)
	}

	controlResponse, err := http.Post(server.URL+"/api/rooms/"+created.RoomID+"/participants/"+created.OwnerID+"/mute", "application/json", nil)
	if err != nil {
		t.Fatalf("participant control request: %v", err)
	}
	defer controlResponse.Body.Close()
	if controlResponse.StatusCode != http.StatusOK {
		t.Fatalf("expected control status 200, got %d", controlResponse.StatusCode)
	}

	statusResponse, err := http.Get(server.URL + "/api/rooms/" + created.RoomID + "/recording/status")
	if err != nil {
		t.Fatalf("recording status request: %v", err)
	}
	defer statusResponse.Body.Close()
	if statusResponse.StatusCode != http.StatusOK {
		t.Fatalf("expected recording status 200, got %d", statusResponse.StatusCode)
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

func TestMeetingEventPersistsConferenceReport(t *testing.T) {
	storagePath := filepath.Join(t.TempDir(), "reports.json")
	cfg := config.Config{TokenTTL: time.Hour, ReportsStoragePath: storagePath}
	handler := NewServer(cfg)

	createdBody := bytes.NewBufferString(`{"roomName":"save-room","userName":"Madi","event":"created"}`)
	createdResponse := httptest.NewRecorder()
	handler.ServeHTTP(createdResponse, httptest.NewRequest(http.MethodPost, "/api/meetings/events", createdBody))
	if createdResponse.Code != http.StatusOK {
		t.Fatalf("expected created status 200, got %d: %s", createdResponse.Code, createdResponse.Body.String())
	}

	var createdPayload struct {
		ReportID string `json:"reportId"`
	}
	if err := json.Unmarshal(createdResponse.Body.Bytes(), &createdPayload); err != nil {
		t.Fatalf("decode created response: %v", err)
	}
	if createdPayload.ReportID == "" {
		t.Fatal("expected report id for created conference")
	}

	leftBody := bytes.NewBufferString(`{"roomName":"save-room","userName":"Madi","event":"left"}`)
	leftResponse := httptest.NewRecorder()
	handler.ServeHTTP(leftResponse, httptest.NewRequest(http.MethodPost, "/api/meetings/events", leftBody))
	if leftResponse.Code != http.StatusOK {
		t.Fatalf("expected left status 200, got %d: %s", leftResponse.Code, leftResponse.Body.String())
	}

	reloadedHandler := NewServer(cfg)
	reportsRecorder := httptest.NewRecorder()
	reloadedHandler.ServeHTTP(reportsRecorder, httptest.NewRequest(http.MethodGet, "/api/reports", nil))
	if reportsRecorder.Code != http.StatusOK {
		t.Fatalf("expected reports status 200, got %d: %s", reportsRecorder.Code, reportsRecorder.Body.String())
	}

	var reports reportsResponse
	if err := json.Unmarshal(reportsRecorder.Body.Bytes(), &reports); err != nil {
		t.Fatalf("decode reports: %v", err)
	}
	found := false
	for _, report := range reports.Reports {
		if report.ID == createdPayload.ReportID && report.ProcessingState == "saved" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected persisted saved report %s, got %#v", createdPayload.ReportID, reports.Reports)
	}
}

func TestConferenceFinalStatusWaitsForEgressProcessing(t *testing.T) {
	server := &Server{
		egress: livekitservice.NewEgressManager(livekitservice.EgressConfig{
			Enabled:   true,
			ServerURL: "ws://livekit:7880",
			APIKey:    "key",
			APISecret: "secret",
			S3: livekitservice.S3Config{
				Bucket: "alemlive-recordings",
				Region: "us-east-1",
			},
		}),
		generatedReportStore: map[string]reportDetailResponse{},
		deletedReportIDs:     map[string]struct{}{},
		activeMeetings:       map[string]meetingSession{},
		latestRoomReports:    map[string]string{},
	}
	now := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)

	created := server.recordConferenceEvent("egress-room", "Madi", "created", now)
	if created.ReportID == "" {
		t.Fatal("expected report id")
	}

	left := server.recordConferenceEvent("egress-room", "Madi", "left", now.Add(time.Minute))
	if left.ConferenceStatus != "processing" {
		t.Fatalf("expected processing final status while egress is configured, got %#v", left)
	}

	detail, ok := server.reportDetailByID(created.ReportID)
	if !ok {
		t.Fatal("expected conference report detail")
	}
	if detail.Report.ProcessingState != "processing" {
		t.Fatalf("expected persisted processing report, got %#v", detail.Report)
	}
}

func TestLeaveWithoutActiveSessionDoesNotCreateReport(t *testing.T) {
	handler := NewServer(config.Config{TokenTTL: time.Hour})

	leaveResponse := httptest.NewRecorder()
	handler.ServeHTTP(leaveResponse, httptest.NewRequest(http.MethodPost, "/api/rooms/ghost-room/leave", bytes.NewBufferString(`{"userName":"Madi"}`)))
	if leaveResponse.Code != http.StatusOK {
		t.Fatalf("expected leave status 200, got %d: %s", leaveResponse.Code, leaveResponse.Body.String())
	}

	var leavePayload map[string]any
	if err := json.Unmarshal(leaveResponse.Body.Bytes(), &leavePayload); err != nil {
		t.Fatalf("decode leave response: %v", err)
	}
	if leavePayload["reportId"] != nil {
		t.Fatalf("stray leave should not create report: %#v", leavePayload)
	}

	reportsRecorder := httptest.NewRecorder()
	handler.ServeHTTP(reportsRecorder, httptest.NewRequest(http.MethodGet, "/api/reports?q=ghost-room", nil))
	if reportsRecorder.Code != http.StatusOK {
		t.Fatalf("expected reports status 200, got %d: %s", reportsRecorder.Code, reportsRecorder.Body.String())
	}
	var reports reportsResponse
	if err := json.Unmarshal(reportsRecorder.Body.Bytes(), &reports); err != nil {
		t.Fatalf("decode reports: %v", err)
	}
	if reports.Total != 0 {
		t.Fatalf("stray leave created a report: %#v", reports.Reports)
	}
}
