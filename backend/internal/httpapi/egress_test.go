package httpapi

import (
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
	if detail.RecordingURL != publicURL {
		t.Fatalf("expected public recording URL, got %#v", detail.RecordingURL)
	}
	if detail.Report.ProcessingState != "ready" {
		t.Fatalf("expected video-ready report even when STT fails, got %#v", detail.Report)
	}
	if len(detail.Summary) == 0 || !strings.Contains(detail.Summary[0].Text, "processing failed") {
		t.Fatalf("expected partial failure summary, got %#v", detail.Summary)
	}
}
