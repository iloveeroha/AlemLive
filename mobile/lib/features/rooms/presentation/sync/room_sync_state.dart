import 'package:alem_live_mobile/features/rooms/domain/entities/recording_status.dart';
import 'package:alem_live_mobile/features/rooms/domain/entities/room_participant.dart';

enum RoomEventsConnectionStatus {
  idle,
  connecting,
  connected,
  reconnecting,
  polling,
  mock,
  disconnected,
  error,
}

enum RoomEventType {
  participantJoined,
  participantLeft,
  participantMicChanged,
  participantCameraChanged,
  participantScreenShareChanged,
  ownerChanged,
  recordingStarted,
  recordingStopped,
  recordingStatusChanged,
  roomClosed,
  unknown,
}

class RoomEventMessage {
  const RoomEventMessage({required this.type, required this.payload});

  factory RoomEventMessage.fromJson(Map<String, dynamic> json) {
    final type = switch (json['type']?.toString()) {
      'participant_joined' => RoomEventType.participantJoined,
      'participant_left' => RoomEventType.participantLeft,
      'participant_mic_changed' => RoomEventType.participantMicChanged,
      'participant_camera_changed' => RoomEventType.participantCameraChanged,
      'participant_screen_share_changed' =>
        RoomEventType.participantScreenShareChanged,
      'owner_changed' => RoomEventType.ownerChanged,
      'recording_started' => RoomEventType.recordingStarted,
      'recording_stopped' => RoomEventType.recordingStopped,
      'recording_status_changed' => RoomEventType.recordingStatusChanged,
      'room_closed' => RoomEventType.roomClosed,
      _ => RoomEventType.unknown,
    };

    final rawPayload = json['payload'];
    final payload =
        rawPayload is Map<String, dynamic>
              ? rawPayload
              : <String, dynamic>{...json}
          ..remove('type');

    return RoomEventMessage(type: type, payload: payload);
  }

  final RoomEventType type;
  final Map<String, dynamic> payload;
}

class RoomSyncState {
  const RoomSyncState({
    required this.connectionStatus,
    required this.participants,
    this.ownerId,
    this.recordingState,
    this.notice,
    this.errorMessage,
    this.roomClosed = false,
  });

  const RoomSyncState.initial()
    : this(
        connectionStatus: RoomEventsConnectionStatus.idle,
        participants: const <String, RoomParticipant>{},
      );

  final RoomEventsConnectionStatus connectionStatus;
  final Map<String, RoomParticipant> participants;
  final String? ownerId;
  final RecordingState? recordingState;
  final String? notice;
  final String? errorMessage;
  final bool roomClosed;

  bool get isRealtimeConnected {
    return connectionStatus == RoomEventsConnectionStatus.connected ||
        connectionStatus == RoomEventsConnectionStatus.mock;
  }

  bool get isPolling => connectionStatus == RoomEventsConnectionStatus.polling;

  RoomSyncState copyWith({
    RoomEventsConnectionStatus? connectionStatus,
    Map<String, RoomParticipant>? participants,
    String? ownerId,
    RecordingState? recordingState,
    String? notice,
    String? errorMessage,
    bool? roomClosed,
    bool clearNotice = false,
    bool clearErrorMessage = false,
  }) {
    return RoomSyncState(
      connectionStatus: connectionStatus ?? this.connectionStatus,
      participants: participants ?? this.participants,
      ownerId: ownerId ?? this.ownerId,
      recordingState: recordingState ?? this.recordingState,
      notice: clearNotice ? null : notice ?? this.notice,
      errorMessage: clearErrorMessage
          ? null
          : errorMessage ?? this.errorMessage,
      roomClosed: roomClosed ?? this.roomClosed,
    );
  }
}
