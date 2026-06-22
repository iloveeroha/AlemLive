package livekit

import (
	"context"
	"strings"

	lkproto "github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
)

func EnsureRoom(ctx context.Context, serverURL, apiKey, apiSecret, roomName string) error {
	serverURL = strings.TrimSpace(serverURL)
	apiKey = strings.TrimSpace(apiKey)
	apiSecret = strings.TrimSpace(apiSecret)
	roomName = strings.TrimSpace(roomName)
	if serverURL == "" || apiKey == "" || apiSecret == "" || roomName == "" {
		return ErrEgressNotConfigured
	}

	client := lksdk.NewRoomServiceClient(serverURL, apiKey, apiSecret)
	_, err := client.CreateRoom(ctx, &lkproto.CreateRoomRequest{
		Name:             roomName,
		EmptyTimeout:     10 * 60,
		DepartureTimeout: 30,
	})
	if err == nil || isAlreadyExistsError(err) {
		return nil
	}
	return err
}

func isAlreadyExistsError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "already") ||
		strings.Contains(message, "exist") ||
		strings.Contains(message, "conflict")
}
