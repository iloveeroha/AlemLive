package httpapi

import (
	"testing"
	"time"

	"github.com/iloveeroha/AlemLive/backend/internal/config"
	livekitservice "github.com/iloveeroha/AlemLive/backend/internal/livekit"
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
