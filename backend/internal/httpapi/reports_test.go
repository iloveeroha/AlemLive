package httpapi

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
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

func TestReportsTypeFilter(t *testing.T) {
	handler := NewServer(config.Config{TokenTTL: time.Hour})
	request := httptest.NewRequest(http.MethodGet, "/api/reports?types=readout", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}

	var payload reportsResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Total != 0 {
		t.Fatalf("expected no readout demo reports, got %#v", payload.Reports)
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

	transcriptRequest := httptest.NewRequest(http.MethodGet, "/api/reports/read-intro/download?format=transcript", nil)
	transcriptResponse := httptest.NewRecorder()
	handler.ServeHTTP(transcriptResponse, transcriptRequest)
	if transcriptResponse.Code != http.StatusOK {
		t.Fatalf("expected transcript status 200, got %d: %s", transcriptResponse.Code, transcriptResponse.Body.String())
	}
	if !strings.Contains(transcriptResponse.Body.String(), "00:") {
		t.Fatalf("transcript download is incomplete: %s", transcriptResponse.Body.String())
	}

	videoRequest := httptest.NewRequest(http.MethodGet, "/api/reports/read-intro/download?format=video", nil)
	videoResponse := httptest.NewRecorder()
	handler.ServeHTTP(videoResponse, videoRequest)
	if videoResponse.Code != http.StatusNotFound {
		t.Fatalf("expected video status 404 without recording, got %d: %s", videoResponse.Code, videoResponse.Body.String())
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
	renamedDetail := httptest.NewRecorder()
	handler.ServeHTTP(renamedDetail, httptest.NewRequest(http.MethodGet, "/api/reports/read-intro", nil))
	if renamedDetail.Code != http.StatusOK {
		t.Fatalf("expected renamed detail status 200, got %d: %s", renamedDetail.Code, renamedDetail.Body.String())
	}
	var renamedPayload reportDetailResponse
	if err := json.Unmarshal(renamedDetail.Body.Bytes(), &renamedPayload); err != nil {
		t.Fatalf("decode renamed detail: %v", err)
	}
	if renamedPayload.Report.Title != "Renamed report" {
		t.Fatalf("rename was not persisted: %#v", renamedPayload.Report)
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
	deletedDetail := httptest.NewRecorder()
	handler.ServeHTTP(deletedDetail, httptest.NewRequest(http.MethodGet, "/api/reports/read-intro", nil))
	if deletedDetail.Code != http.StatusNotFound {
		t.Fatalf("expected deleted detail status 404, got %d: %s", deletedDetail.Code, deletedDetail.Body.String())
	}
}

func TestReportDeletePersistsDemoTombstone(t *testing.T) {
	storagePath := filepath.Join(t.TempDir(), "reports.json")
	cfg := config.Config{TokenTTL: time.Hour, ReportsStoragePath: storagePath}
	handler := NewServer(cfg)

	deleted := httptest.NewRecorder()
	handler.ServeHTTP(deleted, httptest.NewRequest(http.MethodDelete, "/api/reports/read-intro", nil))
	if deleted.Code != http.StatusOK {
		t.Fatalf("expected delete status 200, got %d: %s", deleted.Code, deleted.Body.String())
	}

	reloadedHandler := NewServer(cfg)
	detail := httptest.NewRecorder()
	reloadedHandler.ServeHTTP(detail, httptest.NewRequest(http.MethodGet, "/api/reports/read-intro", nil))
	if detail.Code != http.StatusNotFound {
		t.Fatalf("expected deleted demo report to stay hidden, got %d: %s", detail.Code, detail.Body.String())
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

func TestReportNotesPatch(t *testing.T) {
	handler := NewServer(config.Config{TokenTTL: time.Hour})
	body := bytes.NewBufferString(`{
		"summary":[{"title":"Updated","text":"Saved from frontend"}],
		"actionItems":[{"id":"x","title":"Check notes","task":"Check notes","owner":"Backend","due":"Today","priority":"Medium","status":"open"}]
	}`)
	request := httptest.NewRequest(http.MethodPatch, "/api/reports/read-intro/notes", body)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["status"] != "saved" {
		t.Fatalf("unexpected notes response: %#v", payload)
	}

	getNotes := httptest.NewRecorder()
	handler.ServeHTTP(getNotes, httptest.NewRequest(http.MethodGet, "/api/reports/read-intro/notes", nil))
	if getNotes.Code != http.StatusOK {
		t.Fatalf("expected notes status 200, got %d: %s", getNotes.Code, getNotes.Body.String())
	}
	var notesPayload struct {
		Summary     []summarySection   `json:"summary"`
		ActionItems []reportActionItem `json:"actionItems"`
	}
	if err := json.Unmarshal(getNotes.Body.Bytes(), &notesPayload); err != nil {
		t.Fatalf("decode notes response: %v", err)
	}
	if len(notesPayload.Summary) != 1 || notesPayload.Summary[0].Title != "Updated" {
		t.Fatalf("notes patch was not persisted: %#v", notesPayload)
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

func TestReportRecordingStream(t *testing.T) {
	handler := NewServer(config.Config{TokenTTL: time.Hour})
	metadata := httptest.NewRecorder()
	handler.ServeHTTP(metadata, httptest.NewRequest(http.MethodGet, "/api/reports/read-intro/recording", nil))
	if metadata.Code != http.StatusOK {
		t.Fatalf("expected recording metadata status 200, got %d: %s", metadata.Code, metadata.Body.String())
	}
	var payload struct {
		Available bool `json:"available"`
	}
	if err := json.Unmarshal(metadata.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode recording metadata: %v", err)
	}
	if payload.Available {
		t.Fatalf("demo report should not pretend to have recording: %#v", payload)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/reports/read-intro/recording/stream", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "Recording is not available") {
		t.Fatalf("expected missing recording message, got %s", response.Body.String())
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

func TestReportRecordingUploadRunsInBackground(t *testing.T) {
	sttServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/audio/transcriptions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("unexpected authorization header: %s", got)
		}
		if err := r.ParseMultipartForm(1024 * 1024); err != nil {
			t.Fatalf("parse multipart: %v", err)
		}
		if r.FormValue("model") != "openai/whisper-large-v3" {
			t.Fatalf("unexpected model: %s", r.FormValue("model"))
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"text": "Команда обсудила загрузку записи и фоновую обработку отчета.",
			"segments": []map[string]any{
				{"id": 1, "start": 0, "end": 3, "text": "Команда обсудила загрузку записи."},
				{"id": 2, "start": 3, "end": 6, "text": "Фоновая обработка отчета работает."},
			},
		})
	}))
	defer sttServer.Close()

	handler := NewServer(config.Config{
		TokenTTL:              time.Hour,
		STTBaseURL:            sttServer.URL,
		STTAPIKey:             "test-key",
		STTModel:              "openai/whisper-large-v3",
		STTTimeout:            time.Second,
		LLMTimeout:            time.Second,
		LLMBaseURL:            "",
		LLMAPIKey:             "",
		LLMModel:              "",
		RecordingsStoragePath: t.TempDir(),
	})

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("roomName", "alem-meeting"); err != nil {
		t.Fatalf("write roomName: %v", err)
	}
	if err := writer.WriteField("title", "Async upload"); err != nil {
		t.Fatalf("write title: %v", err)
	}
	file, err := writer.CreateFormFile("file", "meeting.webm")
	if err != nil {
		t.Fatalf("create file: %v", err)
	}
	if _, err := file.Write([]byte("fake audio")); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/reports/upload", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d: %s", response.Code, response.Body.String())
	}

	var uploadPayload struct {
		Report reportRow `json:"report"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &uploadPayload); err != nil {
		t.Fatalf("decode upload response: %v", err)
	}
	if uploadPayload.Report.ProcessingState != "processing" {
		t.Fatalf("expected processing report, got %#v", uploadPayload.Report)
	}

	streamResponse := httptest.NewRecorder()
	handler.ServeHTTP(streamResponse, httptest.NewRequest(http.MethodGet, "/api/reports/"+uploadPayload.Report.ID+"/recording/stream", nil))
	if streamResponse.Code != http.StatusOK {
		t.Fatalf("expected stream status 200, got %d: %s", streamResponse.Code, streamResponse.Body.String())
	}
	if streamResponse.Body.String() != "fake audio" {
		t.Fatalf("unexpected stream body: %q", streamResponse.Body.String())
	}

	var detail reportDetailResponse
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		detailResponse := httptest.NewRecorder()
		handler.ServeHTTP(detailResponse, httptest.NewRequest(http.MethodGet, "/api/reports/"+uploadPayload.Report.ID, nil))
		if detailResponse.Code != http.StatusOK {
			t.Fatalf("expected detail status 200, got %d: %s", detailResponse.Code, detailResponse.Body.String())
		}
		if err := json.Unmarshal(detailResponse.Body.Bytes(), &detail); err != nil {
			t.Fatalf("decode detail response: %v", err)
		}
		if detail.Report.ProcessingState == "ready" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	if detail.Report.ProcessingState != "ready" {
		t.Fatalf("expected ready report after background job, got %#v", detail.Report)
	}
	if len(detail.TranscriptLines) == 0 || len(detail.Summary) == 0 {
		t.Fatalf("expected transcript and summary, got %#v", detail)
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
