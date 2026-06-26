package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/iloveeroha/AlemLive/backend/internal/config"
	livekitservice "github.com/iloveeroha/AlemLive/backend/internal/livekit"
	"github.com/iloveeroha/AlemLive/backend/internal/llm"
)

func TestEgressRecordingURLsUseInternalDownloadAndPublicPlayback(t *testing.T) {
	server := &Server{cfg: config.Config{
		LiveKitS3Endpoint:          "http://minio:9000",
		LiveKitS3Bucket:            "alemlive-recordings",
		LiveKitEgressPublicBaseURL: "http://localhost:9000/alemlive-recordings",
	}}
	state := livekitservice.EgressState{
		RoomName:  "alem-meeting",
		FilePath:  "alemlive/alem-meeting/20260616-120000.mp4",
		PublicURL: "http://localhost:9000/alemlive-recordings/alemlive/alem-meeting/20260616-120000.mp4",
	}

	downloadURL := server.firstRecordingDownloadURL(state, nil)
	if downloadURL != "http://minio:9000/alemlive-recordings/alemlive/alem-meeting/20260616-120000.mp4" {
		t.Fatalf("unexpected download URL: %s", downloadURL)
	}

	playbackURL := server.firstRecordingPlaybackURL(state, nil)
	if playbackURL != state.PublicURL {
		t.Fatalf("unexpected playback URL: %s", playbackURL)
	}
}

func TestEgressRecordingURLsIgnoreMissingFilePath(t *testing.T) {
	server := &Server{cfg: config.Config{
		LiveKitS3Endpoint: "http://minio:9000",
		LiveKitS3Bucket:   "alemlive-recordings",
		TokenTTL:          time.Hour,
	}}

	if got := server.firstRecordingDownloadURL(livekitservice.EgressState{}, nil); got != "" {
		t.Fatalf("expected empty download URL, got %s", got)
	}
}

func TestTrackRecordingURLsUseInternalDownloadAndPublicPlayback(t *testing.T) {
	server := &Server{cfg: config.Config{
		LiveKitS3Endpoint: "http://minio:9000",
		LiveKitS3Bucket:   "alemlive-recordings",
		TokenTTL:          time.Hour,
	}}

	urls := server.trackRecordingDownloadURLs(livekitservice.TrackEgressState{
		FilePath:  "alemlive/abc/audio/asyl-TR_1.ogg",
		PublicURL: "http://localhost:9000/alemlive-recordings/alemlive/abc/audio/asyl-TR_1.ogg",
	})

	if len(urls) != 2 {
		t.Fatalf("expected internal and public URLs, got %#v", urls)
	}
	if urls[0] != "http://minio:9000/alemlive-recordings/alemlive/abc/audio/asyl-TR_1.ogg" {
		t.Fatalf("unexpected internal URL: %#v", urls)
	}
}

func TestProcessEgressRecordingStoresVideoWithoutSTT(t *testing.T) {
	now := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)
	server := &Server{
		cfg: config.Config{
			LiveKitS3Endpoint:          "http://minio:9000",
			LiveKitS3Bucket:            "alemlive-recordings",
			LiveKitEgressPublicBaseURL: "http://localhost:9000/alemlive-recordings",
		},
		clock:                func() time.Time { return now },
		generatedReportStore: map[string]reportDetailResponse{},
		deletedReportIDs:     map[string]struct{}{},
		latestRoomReports:    map[string]string{},
	}

	server.processEgressRecording(t.Context(), livekitservice.EgressState{
		RoomName:  "alem-meeting",
		EgressID:  "egress-id",
		FilePath:  "alemlive/alem-meeting/20260616-120000.mp4",
		PublicURL: "http://localhost:9000/alemlive-recordings/alemlive/alem-meeting/20260616-120000.mp4",
	}, nil)

	detail, ok := server.reportDetailByID("egress-egress-id")
	if !ok {
		t.Fatal("expected egress report to be stored")
	}
	if detail.RecordingURL == "" {
		t.Fatalf("expected recording URL, got %#v", detail)
	}
	if detail.RecordingSourceURL == "" {
		t.Fatalf("expected recording source URL, got %#v", detail)
	}
	if detail.Report.ProcessingState != "ready" {
		t.Fatalf("expected ready report without STT, got %#v", detail.Report)
	}
}

func TestProcessEgressRecordingKeepsVideoWhenSTTFails(t *testing.T) {
	recordingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "video/mp4")
		_, _ = w.Write([]byte("fake video"))
	}))
	defer recordingServer.Close()

	sttServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "stt unavailable", http.StatusBadGateway)
	}))
	defer sttServer.Close()

	now := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)
	server := &Server{
		cfg: config.Config{
			LiveKitS3Endpoint: recordingServer.URL,
			LiveKitS3Bucket:   "alemlive-recordings",
			STTModel:          "openai/whisper-large-v3",
			STTTimeout:        time.Second,
			LLMTimeout:        time.Second,
		},
		stt:                  llm.New(sttServer.URL, "test-key", "openai/whisper-large-v3", time.Second),
		clock:                func() time.Time { return now },
		generatedReportStore: map[string]reportDetailResponse{},
		deletedReportIDs:     map[string]struct{}{},
		latestRoomReports:    map[string]string{},
	}

	publicURL := "http://localhost:9000/alemlive-recordings/alemlive/alem-meeting/20260616-120000.mp4"
	server.processEgressRecording(t.Context(), livekitservice.EgressState{
		RoomName:  "alem-meeting",
		EgressID:  "egress-id",
		FilePath:  "alemlive/alem-meeting/20260616-120000.mp4",
		PublicURL: publicURL,
	}, nil)

	detail, ok := server.reportDetailByID("egress-egress-id")
	if !ok {
		t.Fatal("expected egress report to be stored")
	}
	if detail.RecordingURL != reportStreamURL("egress-egress-id") {
		t.Fatalf("expected backend stream recording URL, got %#v", detail.RecordingURL)
	}
	if detail.RecordingSourceURL == "" || detail.RecordingSourceURL == detail.RecordingURL {
		t.Fatalf("expected internal recording source URL, got %#v", detail)
	}
	if detail.Report.ProcessingState != "ready" {
		t.Fatalf("expected video-ready report even when STT fails, got %#v", detail.Report)
	}
	if len(detail.Summary) == 0 || !strings.Contains(detail.Summary[0].Text, "processing failed") {
		t.Fatalf("expected partial failure summary, got %#v", detail.Summary)
	}
}

func TestProcessEgressRecordingMarksMissingFileAsRecordingFailed(t *testing.T) {
	recordingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer recordingServer.Close()

	now := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)
	server := &Server{
		cfg: config.Config{
			LiveKitS3Endpoint: recordingServer.URL,
			LiveKitS3Bucket:   "alemlive-recordings",
			STTModel:          "openai/whisper-large-v3",
			STTTimeout:        time.Second,
			LLMTimeout:        time.Second,
		},
		stt:                  llm.New("http://stt.invalid", "test-key", "openai/whisper-large-v3", time.Second),
		clock:                func() time.Time { return now },
		generatedReportStore: map[string]reportDetailResponse{},
		deletedReportIDs:     map[string]struct{}{},
		latestRoomReports:    map[string]string{},
	}

	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()

	server.processEgressRecording(ctx, livekitservice.EgressState{
		RoomName: "alem-meeting",
		EgressID: "egress-id",
		FilePath: "alemlive/alem-meeting/20260616-120000.mp4",
	}, nil)

	detail, ok := server.reportDetailByID("egress-egress-id")
	if !ok {
		t.Fatal("expected egress report to be stored")
	}
	if detail.Report.RecordingStatus != "failed" {
		t.Fatalf("expected recording failed, got %#v", detail.Report)
	}
	if detail.Report.TranscriptionStatus != "not_started" || detail.Report.AnalysisStatus != "not_started" {
		t.Fatalf("expected downstream stages not to start, got %#v", detail.Report)
	}
	if detail.LastError == "" || detail.RecordingError == "" {
		t.Fatalf("expected recording error to be saved, got %#v", detail)
	}
}

func TestDownloadRecordingWithRetryWaitsForObject(t *testing.T) {
	calls := 0
	recordingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			http.Error(w, "not ready", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "video/mp4")
		_, _ = w.Write([]byte("video bytes"))
	}))
	defer recordingServer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	fileName, contentType, data, _, err := downloadRecordingWithRetry(ctx, []string{recordingServer.URL + "/recording.mp4"}, time.Second)
	if err != nil {
		t.Fatalf("expected retry to succeed, got %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected two attempts, got %d", calls)
	}
	if fileName != "recording.mp4" || contentType != "video/mp4" || string(data) != "video bytes" {
		t.Fatalf("unexpected recording download: %s %s %q", fileName, contentType, string(data))
	}
}
