package livekit

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestGenerateToken(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)

	token, expiresAt, err := GenerateToken("api-key", "secret", "Madi", "alem-meeting", time.Hour, now)
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}

	if expiresAt.Unix() != now.Add(time.Hour).Unix() {
		t.Fatalf("unexpected expiration: %v", expiresAt)
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("token should have 3 parts, got %d", len(parts))
	}

	unsigned := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, []byte("secret"))
	_, _ = mac.Write([]byte(unsigned))
	expectedSignature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(parts[2]), []byte(expectedSignature)) {
		t.Fatal("signature does not match")
	}

	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	if payload["iss"] != "api-key" || payload["sub"] != "Madi" {
		t.Fatalf("unexpected identity claims: %#v", payload)
	}

	video, ok := payload["video"].(map[string]any)
	if !ok {
		t.Fatalf("video grant missing: %#v", payload)
	}
	if video["room"] != "alem-meeting" || video["roomJoin"] != true {
		t.Fatalf("unexpected video grant: %#v", video)
	}
}

func TestGenerateTokenRequiresInputs(t *testing.T) {
	_, _, err := GenerateToken("", "secret", "user", "room", time.Hour, time.Now())
	if err == nil {
		t.Fatal("expected missing api key error")
	}
}
