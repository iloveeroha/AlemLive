package livekit

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type VideoGrant struct {
	Room           string `json:"room,omitempty"`
	RoomJoin       bool   `json:"roomJoin,omitempty"`
	CanPublish     bool   `json:"canPublish,omitempty"`
	CanSubscribe   bool   `json:"canSubscribe,omitempty"`
	CanPublishData bool   `json:"canPublishData,omitempty"`
}

type claims struct {
	Issuer    string     `json:"iss"`
	Subject   string     `json:"sub"`
	NotBefore int64      `json:"nbf"`
	ExpiresAt int64      `json:"exp"`
	Name      string     `json:"name,omitempty"`
	Metadata  string     `json:"metadata,omitempty"`
	Video     VideoGrant `json:"video"`
}

func GenerateToken(apiKey, secret, identity, room, metadata string, ttl time.Duration, now time.Time) (string, time.Time, error) {
	return GenerateTokenWithName(apiKey, secret, identity, identity, room, metadata, ttl, now)
}

func GenerateTokenWithName(apiKey, secret, identity, name, room, metadata string, ttl time.Duration, now time.Time) (string, time.Time, error) {
	apiKey = strings.TrimSpace(apiKey)
	secret = strings.TrimSpace(secret)
	identity = strings.TrimSpace(identity)
	name = strings.TrimSpace(name)
	room = strings.TrimSpace(room)

	if apiKey == "" {
		return "", time.Time{}, errors.New("livekit api key is required")
	}
	if secret == "" {
		return "", time.Time{}, errors.New("livekit api secret is required")
	}
	if identity == "" {
		return "", time.Time{}, errors.New("identity is required")
	}
	if room == "" {
		return "", time.Time{}, errors.New("room is required")
	}
	if ttl <= 0 {
		return "", time.Time{}, errors.New("ttl must be positive")
	}

	expiresAt := now.Add(ttl)
	body := claims{
		Issuer:    apiKey,
		Subject:   identity,
		NotBefore: now.Unix(),
		ExpiresAt: expiresAt.Unix(),
		Name:      firstNonEmptyToken(name, identity),
		Metadata:  metadata,
		Video: VideoGrant{
			Room:           room,
			RoomJoin:       true,
			CanPublish:     true,
			CanSubscribe:   true,
			CanPublishData: true,
		},
	}

	token, err := signJWT(body, []byte(secret))
	if err != nil {
		return "", time.Time{}, err
	}

	return token, expiresAt, nil
}

func firstNonEmptyToken(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func signJWT(payload claims, secret []byte) (string, error) {
	header := map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	encoder := base64.RawURLEncoding
	unsigned := encoder.EncodeToString(headerJSON) + "." + encoder.EncodeToString(payloadJSON)

	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(unsigned))

	return unsigned + "." + encoder.EncodeToString(mac.Sum(nil)), nil
}
