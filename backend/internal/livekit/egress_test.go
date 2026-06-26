package livekit

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestLikelyStopAcceptedError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "livekit rpc timeout", err: errors.New("twirp error unknown: request timed out"), want: true},
		{name: "deadline code", err: errors.New("code = deadline_exceeded desc = request timed out"), want: true},
		{name: "context deadline", err: errors.New("context deadline exceeded"), want: true},
		{name: "real api error", err: errors.New("permission denied"), want: false},
		{name: "nil", err: nil, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isLikelyStopAcceptedError(tt.err); got != tt.want {
				t.Fatalf("isLikelyStopAcceptedError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTrackRecordingPathUsesParticipantAndTrack(t *testing.T) {
	manager := NewEgressManager(EgressConfig{FilePrefix: "alemlive"})
	now := time.Date(2026, 6, 26, 10, 30, 0, 0, time.UTC)

	got := manager.trackRecordingPath("abc-defg-hij", ParticipantTrack{
		ParticipantName: "Asyl Asyl",
		TrackID:         "TR_audio_123",
	}, now)

	if !strings.Contains(got, "alemlive/abc-defg-hij/20260626-103000/audio/asyl-asyl-tr_audio_123.ogg") {
		t.Fatalf("unexpected track recording path: %s", got)
	}
}

func TestNormalizeParticipantTracksDeduplicatesTrackIDs(t *testing.T) {
	tracks := normalizeParticipantTracks([]ParticipantTrack{
		{ParticipantName: "A", TrackID: "TR_1"},
		{ParticipantName: "A duplicate", TrackID: "TR_1"},
		{ParticipantName: "B", TrackID: "TR_2"},
		{ParticipantName: "empty"},
	})

	if len(tracks) != 2 {
		t.Fatalf("expected 2 unique tracks, got %#v", tracks)
	}
	if tracks[0].ParticipantName != "A" || tracks[1].TrackID != "TR_2" {
		t.Fatalf("unexpected normalized tracks: %#v", tracks)
	}
}
