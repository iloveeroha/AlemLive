import 'package:alem_live_mobile/features/rooms/domain/entities/chat_message.dart';
import 'package:alem_live_mobile/features/rooms/domain/entities/recording_status.dart';
import 'package:alem_live_mobile/features/rooms/domain/entities/room_participant.dart';
import 'package:livekit_client/livekit_client.dart';

enum LiveKitRoomStatus {
  mock,
  connecting,
  connected,
  reconnecting,
  disconnected,
  error,
}

enum ParticipantControlType { microphone, camera }

class ParticipantControlKey {
  const ParticipantControlKey({
    required this.participantId,
    required this.type,
  });

  final String participantId;
  final ParticipantControlType type;

  @override
  bool operator ==(Object other) {
    return other is ParticipantControlKey &&
        other.participantId == participantId &&
        other.type == type;
  }

  @override
  int get hashCode => Object.hash(participantId, type);
}

class RoomParticipantView {
  const RoomParticipantView({required this.participant, this.videoTrack});

  final RoomParticipant participant;
  final VideoTrack? videoTrack;

  bool get canRenderVideo {
    return (participant.isCameraEnabled || participant.isScreenSharing) &&
        videoTrack != null;
  }
}

class LiveKitRoomState {
  const LiveKitRoomState({
    required this.status,
    required this.participants,
    required this.micEnabled,
    required this.cameraEnabled,
    required this.screenSharing,
    required this.recordingState,
    required this.messages,
    required this.controlLoading,
    this.ownerId,
    this.currentUserId,
    this.errorMessage,
    this.controlErrorMessage,
    this.roomNotice,
  });

  factory LiveKitRoomState.initial({
    required bool micEnabled,
    required bool cameraEnabled,
    required String? ownerId,
    required String? currentUserId,
  }) {
    return LiveKitRoomState(
      status: LiveKitRoomStatus.connecting,
      participants: const [],
      micEnabled: micEnabled,
      cameraEnabled: cameraEnabled,
      screenSharing: false,
      recordingState: RecordingState.idle,
      messages: const [],
      controlLoading: const <ParticipantControlKey>{},
      ownerId: ownerId,
      currentUserId: currentUserId,
    );
  }

  final LiveKitRoomStatus status;
  final List<RoomParticipantView> participants;
  final bool micEnabled;
  final bool cameraEnabled;
  final bool screenSharing;
  final RecordingState recordingState;
  final List<ChatMessage> messages;
  final Set<ParticipantControlKey> controlLoading;
  final String? ownerId;
  final String? currentUserId;
  final String? errorMessage;
  final String? controlErrorMessage;
  final String? roomNotice;

  bool get isLoading => status == LiveKitRoomStatus.connecting;
  bool get isConnected => status == LiveKitRoomStatus.connected;
  bool get isMock => status == LiveKitRoomStatus.mock;
  bool get hasError => status == LiveKitRoomStatus.error;
  bool get isRecording => recordingState == RecordingState.recording;
  bool get isCurrentUserOwner {
    if (ownerId == null || currentUserId == null) {
      return participants.any(
        (view) => view.participant.isCurrentUser && view.participant.isOwner,
      );
    }
    return ownerId == currentUserId;
  }

  bool isControlLoading(String participantId, ParticipantControlType type) {
    return controlLoading.contains(
      ParticipantControlKey(participantId: participantId, type: type),
    );
  }

  LiveKitRoomState copyWith({
    LiveKitRoomStatus? status,
    List<RoomParticipantView>? participants,
    bool? micEnabled,
    bool? cameraEnabled,
    bool? screenSharing,
    RecordingState? recordingState,
    List<ChatMessage>? messages,
    Set<ParticipantControlKey>? controlLoading,
    String? ownerId,
    String? currentUserId,
    String? errorMessage,
    String? controlErrorMessage,
    String? roomNotice,
    bool clearErrorMessage = false,
    bool clearControlErrorMessage = false,
    bool clearRoomNotice = false,
  }) {
    return LiveKitRoomState(
      status: status ?? this.status,
      participants: participants ?? this.participants,
      micEnabled: micEnabled ?? this.micEnabled,
      cameraEnabled: cameraEnabled ?? this.cameraEnabled,
      screenSharing: screenSharing ?? this.screenSharing,
      recordingState: recordingState ?? this.recordingState,
      messages: messages ?? this.messages,
      controlLoading: controlLoading ?? this.controlLoading,
      ownerId: ownerId ?? this.ownerId,
      currentUserId: currentUserId ?? this.currentUserId,
      errorMessage: clearErrorMessage
          ? null
          : errorMessage ?? this.errorMessage,
      controlErrorMessage: clearControlErrorMessage
          ? null
          : controlErrorMessage ?? this.controlErrorMessage,
      roomNotice: clearRoomNotice ? null : roomNotice ?? this.roomNotice,
    );
  }
}
