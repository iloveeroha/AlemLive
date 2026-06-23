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

func MuteParticipantTracks(ctx context.Context, serverURL, apiKey, apiSecret, roomName, participantID string, source lkproto.TrackSource, muted bool) error {
	serverURL = strings.TrimSpace(serverURL)
	apiKey = strings.TrimSpace(apiKey)
	apiSecret = strings.TrimSpace(apiSecret)
	roomName = strings.TrimSpace(roomName)
	participantID = strings.TrimSpace(participantID)
	if serverURL == "" || apiKey == "" || apiSecret == "" || roomName == "" || participantID == "" {
		return ErrEgressNotConfigured
	}

	client := lksdk.NewRoomServiceClient(serverURL, apiKey, apiSecret)
	participants, err := client.ListParticipants(ctx, &lkproto.ListParticipantsRequest{Room: roomName})
	if err != nil {
		return err
	}

	mutedAny := false
	for _, participant := range participants.GetParticipants() {
		if participant.GetIdentity() != participantID {
			continue
		}
		for _, track := range participant.GetTracks() {
			if track.GetSource() != source || track.GetSid() == "" {
				continue
			}
			if _, err := client.MutePublishedTrack(ctx, &lkproto.MuteRoomTrackRequest{
				Room:     roomName,
				Identity: participantID,
				TrackSid: track.GetSid(),
				Muted:    muted,
			}); err != nil {
				return err
			}
			mutedAny = true
		}
		break
	}
	if !mutedAny {
		return nil
	}
	return nil
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
