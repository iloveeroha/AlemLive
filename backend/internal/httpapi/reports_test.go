package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/iloveeroha/AlemLive/backend/internal/config"
)

func TestReportsList(t *testing.T) {
	handler := NewServer(config.Config{TokenTTL: time.Hour})
	request := httptest.NewRequest(http.MethodGet, "/api/reports", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}

	var payload reportsResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Total == 0 || len(payload.Reports) == 0 {
		t.Fatalf("expected reports, got %#v", payload)
	}
	if len(payload.Items) != len(payload.Reports) {
		t.Fatalf("items alias should match reports")
	}
	if len(payload.Filters.QuickDateOptions) == 0 {
		t.Fatal("expected date filter options")
	}
}

func TestReportsSearchAndDateFilter(t *testing.T) {
	handler := NewServer(config.Config{TokenTTL: time.Hour})
	request := httptest.NewRequest(http.MethodGet, "/api/reports?q=Copilot&from=2026-01-01&to=2026-01-03", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}

	var payload reportsResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Total != 1 {
		t.Fatalf("expected one filtered report, got %#v", payload)
	}
	if payload.Reports[0].ID != "copilot-search" {
		t.Fatalf("unexpected report: %#v", payload.Reports[0])
	}
}

func TestReportFilters(t *testing.T) {
	handler := NewServer(config.Config{TokenTTL: time.Hour})
	request := httptest.NewRequest(http.MethodGet, "/api/reports/filters", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}

	var payload reportFilters
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Sources) == 0 || len(payload.Folders) == 0 || len(payload.Owners) == 0 {
		t.Fatalf("filters are incomplete: %#v", payload)
	}
}

func TestReportDetail(t *testing.T) {
	handler := NewServer(config.Config{TokenTTL: time.Hour})
	request := httptest.NewRequest(http.MethodGet, "/api/reports/read-intro", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}

	var payload reportDetailResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Report.ID != "read-intro" {
		t.Fatalf("unexpected report id: %s", payload.Report.ID)
	}
	if payload.Report.Participants != 4 {
		t.Fatalf("expected numeric participants, got %d", payload.Report.Participants)
	}
	if len(payload.Summary) == 0 || len(payload.ActionItems) == 0 || len(payload.TranscriptLines) == 0 {
		t.Fatalf("detail payload is incomplete: %#v", payload)
	}
	if len(payload.AIQuestions) == 0 || len(payload.Chapters) == 0 || len(payload.Highlights) == 0 {
		t.Fatalf("ai report payload is incomplete: %#v", payload)
	}
	if payload.Highlights[0].Note == "" || payload.Chapters[0].Duration == "" || payload.SpeakerStats[0].Pace == "" {
		t.Fatalf("frontend detail fields are incomplete: %#v", payload)
	}
}

func TestReportAnalysisAlias(t *testing.T) {
	handler := NewServer(config.Config{TokenTTL: time.Hour})
	request := httptest.NewRequest(http.MethodGet, "/api/reports/read-intro/analysis", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}

	var payload meetingAnalysis
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.RoomName != "read-intro" {
		t.Fatalf("unexpected room name: %s", payload.RoomName)
	}
	if len(payload.Insights.TalkTime) == 0 {
		t.Fatal("expected converted talk time metrics")
	}
}

func TestReportDownload(t *testing.T) {
	handler := NewServer(config.Config{TokenTTL: time.Hour})
	request := httptest.NewRequest(http.MethodGet, "/api/reports/read-intro/download", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	if got := response.Header().Get("Content-Type"); !strings.Contains(got, "text/plain") {
		t.Fatalf("expected text download, got %s", got)
	}
	if !strings.Contains(response.Body.String(), "Action items") {
		t.Fatalf("download body is incomplete: %s", response.Body.String())
	}
}

func TestReportShare(t *testing.T) {
	handler := NewServer(config.Config{TokenTTL: time.Hour})
	request := httptest.NewRequest(http.MethodPost, "/api/reports/read-intro/share", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}

	var payload map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["url"] != "/report/read-intro" {
		t.Fatalf("unexpected share payload: %#v", payload)
	}
}

func TestReportRenameDeleteAndSend(t *testing.T) {
	handler := NewServer(config.Config{TokenTTL: time.Hour})

	rename := httptest.NewRecorder()
	handler.ServeHTTP(rename, httptest.NewRequest(http.MethodPatch, "/api/reports/read-intro", bytes.NewBufferString(`{"title":"Renamed report"}`)))
	if rename.Code != http.StatusOK {
		t.Fatalf("expected rename status 200, got %d: %s", rename.Code, rename.Body.String())
	}

	send := httptest.NewRecorder()
	handler.ServeHTTP(send, httptest.NewRequest(http.MethodPost, "/api/reports/read-intro/send", nil))
	if send.Code != http.StatusOK {
		t.Fatalf("expected send status 200, got %d: %s", send.Code, send.Body.String())
	}

	deleted := httptest.NewRecorder()
	handler.ServeHTTP(deleted, httptest.NewRequest(http.MethodDelete, "/api/reports/read-intro", nil))
	if deleted.Code != http.StatusOK {
		t.Fatalf("expected delete status 200, got %d: %s", deleted.Code, deleted.Body.String())
	}
}

func TestReportUtilityEndpoints(t *testing.T) {
	handler := NewServer(config.Config{TokenTTL: time.Hour})

	copyResponse := httptest.NewRecorder()
	handler.ServeHTTP(copyResponse, httptest.NewRequest(http.MethodGet, "/api/reports/read-intro/copy", nil))
	if copyResponse.Code != http.StatusOK {
		t.Fatalf("expected copy status 200, got %d: %s", copyResponse.Code, copyResponse.Body.String())
	}

	searchResponse := httptest.NewRecorder()
	handler.ServeHTTP(searchResponse, httptest.NewRequest(http.MethodGet, "/api/reports/read-intro/search?q=backend", nil))
	if searchResponse.Code != http.StatusOK {
		t.Fatalf("expected search status 200, got %d: %s", searchResponse.Code, searchResponse.Body.String())
	}
	var searchPayload struct {
		Results []reportSearchResult `json:"results"`
	}
	if err := json.Unmarshal(searchResponse.Body.Bytes(), &searchPayload); err != nil {
		t.Fatalf("decode search response: %v", err)
	}
	if len(searchPayload.Results) == 0 {
		t.Fatal("expected search results")
	}

	promptsResponse := httptest.NewRecorder()
	handler.ServeHTTP(promptsResponse, httptest.NewRequest(http.MethodGet, "/api/reports/read-intro/prompts", nil))
	if promptsResponse.Code != http.StatusOK {
		t.Fatalf("expected prompts status 200, got %d: %s", promptsResponse.Code, promptsResponse.Body.String())
	}

	historyResponse := httptest.NewRecorder()
	handler.ServeHTTP(historyResponse, httptest.NewRequest(http.MethodGet, "/api/reports/read-intro/history", nil))
	if historyResponse.Code != http.StatusOK {
		t.Fatalf("expected history status 200, got %d: %s", historyResponse.Code, historyResponse.Body.String())
	}
}

func TestReportActionsTabsAndSubresources(t *testing.T) {
	handler := NewServer(config.Config{TokenTTL: time.Hour})

	for _, path := range []string{
		"/api/reports/read-intro/actions",
		"/api/reports/read-intro/recording",
		"/api/reports/read-intro/tabs",
		"/api/reports/read-intro/notes",
		"/api/reports/read-intro/action-items",
		"/api/reports/read-intro/transcript",
		"/api/reports/read-intro/deep-dive",
		"/api/reports/read-intro/highlights",
		"/api/reports/read-intro/chapters",
	} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		response := httptest.NewRecorder()

		handler.ServeHTTP(response, request)

		if response.Code != http.StatusOK {
			t.Fatalf("%s expected status 200, got %d: %s", path, response.Code, response.Body.String())
		}
	}
}

func TestReportChatFallback(t *testing.T) {
	handler := NewServer(config.Config{TokenTTL: time.Hour})
	request := httptest.NewRequest(http.MethodPost, "/api/reports/read-intro/chat", bytes.NewBufferString(`{"message":"Какие задачи?"}`))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}

	var payload aiChatResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !strings.Contains(payload.Answer, "Задачи") {
		t.Fatalf("unexpected fallback answer: %q", payload.Answer)
	}
}

func TestReportUpload(t *testing.T) {
	handler := NewServer(config.Config{TokenTTL: time.Hour})
	body := bytes.NewBufferString(`{"title":"Demo upload","owner":"Backend"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/reports/upload", body)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d: %s", response.Code, response.Body.String())
	}

	var payload struct {
		Report reportRow `json:"report"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Report.Title != "Demo upload" || payload.Report.ProcessingState != "processing" {
		t.Fatalf("unexpected upload response: %#v", payload)
	}
}

func TestReportNotFound(t *testing.T) {
	handler := NewServer(config.Config{TokenTTL: time.Hour})
	request := httptest.NewRequest(http.MethodGet, "/api/reports/missing", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", response.Code)
	}
}
