import 'package:alem_live_mobile/features/rooms/domain/entities/recording_status.dart';
import 'package:alem_live_mobile/features/rooms/domain/entities/room_participant.dart';
import 'package:alem_live_mobile/features/rooms/domain/entities/room_session.dart';

abstract interface class RoomsRepository {
  Future<RoomSession> createRoom({
    required String roomName,
    required bool initialMicEnabled,
    required bool initialCameraEnabled,
  });

  Future<RoomSession> joinRoom({required String roomName});

  Future<void> leaveRoom({required String roomId});

  Future<RoomSession?> leaveRoomWithInfo({required String roomId});

  Future<RoomSession> roomInfo({required String roomId});

  Future<List<RoomParticipant>> participants({required String roomId});

  Future<RecordingStatus> startRecording({required String roomId});

  Future<RecordingStatus> stopRecording({required String roomId});

  Future<RecordingStatus> recordingStatus({required String roomId});

  Future<void> transferOwner({
    required String roomId,
    required String participantId,
  });

  Future<void> muteParticipant({
    required String roomId,
    required String participantId,
  });

  Future<void> unmuteParticipant({
    required String roomId,
    required String participantId,
  });

  Future<void> cameraOff({
    required String roomId,
    required String participantId,
  });

  Future<void> cameraOnRequest({
    required String roomId,
    required String participantId,
  });

  Future<void> screenShareStarted({
    required String roomId,
    required String participantId,
  });

  Future<void> screenShareStopped({
    required String roomId,
    required String participantId,
  });
}
