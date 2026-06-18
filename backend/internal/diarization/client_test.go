package diarization

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDiarizeSendsMultipartAndParsesSegments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/diarize" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("unexpected authorization: %s", got)
		}
		if err := r.ParseMultipartForm(1024 * 1024); err != nil {
			t.Fatalf("parse multipart: %v", err)
		}
		if r.FormValue("participants") != "Madi, Aida" {
			t.Fatalf("unexpected participants: %q", r.FormValue("participants"))
		}
		if _, _, err := r.FormFile("file"); err != nil {
			t.Fatalf("expected uploaded file: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"segments": []map[string]any{
				{"start": 0, "end": 2.5, "speaker_label": "SPEAKER_00"},
				{"start": 2.5, "end": 5, "label": "SPEAKER_01"},
			},
		})
	}))
	defer server.Close()

	client := New(server.URL, "test-key", time.Second)
	result, err := client.Diarize(context.Background(), "meeting.webm", "audio/webm", []byte("audio"), Options{
		Participants: "Madi, Aida",
	})
	if err != nil {
		t.Fatalf("diarize: %v", err)
	}
	if len(result.Segments) != 2 {
		t.Fatalf("expected two segments, got %#v", result)
	}
	if result.Segments[0].Speaker != "SPEAKER_00" || result.Segments[1].Speaker != "SPEAKER_01" {
		t.Fatalf("unexpected speakers: %#v", result.Segments)
	}
}

func TestDiarizeRequiresConfiguredClient(t *testing.T) {
	_, err := New("", "", time.Second).Diarize(context.Background(), "a.wav", "audio/wav", []byte("x"), Options{})
	if err == nil || !strings.Contains(err.Error(), ErrNotConfigured.Error()) {
		t.Fatalf("expected not configured error, got %v", err)
	}
}
