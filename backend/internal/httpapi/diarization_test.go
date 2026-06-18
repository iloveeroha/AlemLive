package httpapi

import (
	"testing"

	"github.com/iloveeroha/AlemLive/backend/internal/diarization"
)

func TestApplyDiarizationSegmentsAssignsSpeakersByOverlap(t *testing.T) {
	lines := []transcriptLine{
		{Time: "00:00", Speaker: "Speaker", Text: "Hello", Start: 0, End: 2},
		{Time: "00:02", Speaker: "Speaker", Text: "Hi", Start: 2, End: 4},
		{Time: "00:04", Speaker: "Speaker", Text: "Let's start", Start: 4, End: 6},
	}
	segments := []diarization.Segment{
		{Start: 0, End: 2.2, Speaker: "SPEAKER_00"},
		{Start: 2.2, End: 6, Speaker: "SPEAKER_01"},
	}

	got := applyDiarizationSegments(lines, segments)

	if got[0].Speaker != "Speaker 1" {
		t.Fatalf("first line speaker = %q", got[0].Speaker)
	}
	if got[1].Speaker != "Speaker 2" || got[2].Speaker != "Speaker 2" {
		t.Fatalf("remaining speakers = %q, %q", got[1].Speaker, got[2].Speaker)
	}
}

func TestApplyDiarizationSegmentsKeepsFallbackWithoutMatch(t *testing.T) {
	lines := []transcriptLine{
		{Time: "00:10", Speaker: "Speaker", Text: "Outside segment", Start: 10, End: 12},
	}
	segments := []diarization.Segment{
		{Start: 0, End: 2, Speaker: "SPEAKER_00"},
	}

	got := applyDiarizationSegments(lines, segments)

	if got[0].Speaker != "Speaker" {
		t.Fatalf("expected fallback speaker, got %q", got[0].Speaker)
	}
}

func TestApplyDiarizationSegmentsMapsKnownParticipants(t *testing.T) {
	lines := []transcriptLine{
		{Time: "00:00", Speaker: "Speaker", Text: "Welcome everyone", Start: 0, End: 2},
		{Time: "00:02", Speaker: "Speaker", Text: "Hi, I'm Kelcey and I'm on the CS team", Start: 2, End: 4},
	}
	segments := []diarization.Segment{
		{Start: 0, End: 2, Speaker: "SPEAKER_00"},
		{Start: 2, End: 4, Speaker: "SPEAKER_01"},
	}

	got := applyDiarizationSegments(lines, segments, "Alison Barker, Kelcey Hawthorne, +1 more")

	if got[0].Speaker != "Alison Barker" {
		t.Fatalf("first speaker = %q", got[0].Speaker)
	}
	if got[1].Speaker != "Kelcey Hawthorne" {
		t.Fatalf("self-introduced speaker = %q", got[1].Speaker)
	}
}

func TestApplyDiarizationSegmentsUsesSelfIntroducedNameWithoutParticipantList(t *testing.T) {
	lines := []transcriptLine{
		{Time: "00:00", Speaker: "Speaker", Text: "Hi, I'm Zoe and I manage the product team", Start: 0, End: 2},
	}
	segments := []diarization.Segment{
		{Start: 0, End: 2, Speaker: "SPEAKER_00"},
	}

	got := applyDiarizationSegments(lines, segments)

	if got[0].Speaker != "Zoe" {
		t.Fatalf("speaker = %q", got[0].Speaker)
	}
}
