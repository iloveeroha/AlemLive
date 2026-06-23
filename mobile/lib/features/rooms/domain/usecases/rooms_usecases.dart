import 'package:alem_live_mobile/features/rooms/data/rooms_repository_impl.dart';
import 'package:alem_live_mobile/features/rooms/domain/entities/recording_status.dart';
import 'package:alem_live_mobile/features/rooms/domain/entities/room_participant.dart';
import 'package:alem_live_mobile/features/rooms/domain/entities/room_session.dart';
import 'package:alem_live_mobile/features/rooms/domain/rooms_repository.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

final roomsUseCasesProvider = Provider<RoomsUseCases>((ref) {
  return RoomsUseCases(ref.watch(roomsRepositoryProvider));
});

class RoomsUseCases {
  const RoomsUseCases(this._repository);

  final RoomsRepository _repository;

  Future<RoomSession> createRoom({
    required String roomName,
    required bool initialMicEnabled,
    required bool initialCameraEnabled,
  }) {
    return _repository.createRoom(
      roomName: roomName,
      initialMicEnabled: initialMicEnabled,
      initialCameraEnabled: initialCameraEnabled,
    );
  }

  Future<RoomSession> joinRoom({required String roomName}) {
    return _repository.joinRoom(roomName: roomName);
  }

  Future<void> leaveRoom({required String roomId}) {
    return _repository.leaveRoom(roomId: roomId);
  }

  Future<RoomSession?> leaveRoomWithInfo({required String roomId}) {
    return _repository.leaveRoomWithInfo(roomId: roomId);
  }

  Future<RoomSession> roomInfo({required String roomId}) {
    return _repository.roomInfo(roomId: roomId);
  }

  Future<List<RoomParticipant>> participants({required String roomId}) {
    return _repository.participants(roomId: roomId);
  }

  Future<RecordingStatus> startRecording({required String roomId}) {
    return _repository.startRecording(roomId: roomId);
  }

  Future<RecordingStatus> stopRecording({required String roomId}) {
    return _repository.stopRecording(roomId: roomId);
  }

  Future<RecordingStatus> recordingStatus({required String roomId}) {
    return _repository.recordingStatus(roomId: roomId);
  }

  Future<void> transferOwner({
    required String roomId,
    required String participantId,
  }) {
    return _repository.transferOwner(
      roomId: roomId,
      participantId: participantId,
    );
  }

  Future<void> muteParticipant({
    required String roomId,
    required String participantId,
  }) {
    return _repository.muteParticipant(
      roomId: roomId,
      participantId: participantId,
    );
  }

  Future<void> unmuteParticipant({
    required String roomId,
    required String participantId,
  }) {
    return _repository.unmuteParticipant(
      roomId: roomId,
      participantId: participantId,
    );
  }

  Future<void> cameraOff({
    required String roomId,
    required String participantId,
  }) {
    return _repository.cameraOff(roomId: roomId, participantId: participantId);
  }

  Future<void> cameraOnRequest({
    required String roomId,
    required String participantId,
  }) {
    return _repository.cameraOnRequest(
      roomId: roomId,
      participantId: participantId,
    );
  }

  Future<void> screenShareStarted({
    required String roomId,
    required String participantId,
  }) {
    return _repository.screenShareStarted(
      roomId: roomId,
      participantId: participantId,
    );
  }

  Future<void> screenShareStopped({
    required String roomId,
    required String participantId,
  }) {
    return _repository.screenShareStopped(
      roomId: roomId,
      participantId: participantId,
    );
  }
}
