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

func TestApplyParticipantSpeakerNamesReplacesUnknownLabelsWithParticipants(t *testing.T) {
	lines := []transcriptLine{
		{Time: "01:25", Speaker: "Аленмитинг Три", Text: "Нет, в основном нет.", Start: 85, End: 91},
		{Time: "02:22", Speaker: "Speaker 1", Text: "Тест номер.", Start: 142, End: 149},
	}

	got := applyParticipantSpeakerNames(lines, "Мади Орысбек, Айдана Сейт, Елиас, +1 больше")

	if got[0].Speaker != "Мади Орысбек" {
		t.Fatalf("unknown speaker label = %q", got[0].Speaker)
	}
	if got[1].Speaker != "Айдана Сейт" {
		t.Fatalf("generic speaker label = %q", got[1].Speaker)
	}
}

func TestApplyParticipantSpeakerNamesKeepsKnownParticipantLabels(t *testing.T) {
	lines := []transcriptLine{
		{Time: "00:00", Speaker: "Мади", Text: "Привет", Start: 0, End: 2},
		{Time: "00:03", Speaker: "Айдана Сейт", Text: "Салам", Start: 3, End: 5},
	}

	got := applyParticipantSpeakerNames(lines, "Мади Орысбек, Айдана Сейт")

	if got[0].Speaker != "Мади Орысбек" {
		t.Fatalf("first-name speaker label = %q", got[0].Speaker)
	}
	if got[1].Speaker != "Айдана Сейт" {
		t.Fatalf("known speaker label = %q", got[1].Speaker)
	}
}
