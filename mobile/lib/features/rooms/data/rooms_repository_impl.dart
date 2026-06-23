import 'package:alem_live_mobile/app/config.dart';
import 'package:alem_live_mobile/core/network/dio_client.dart';
import 'package:alem_live_mobile/features/rooms/data/models/room_participant_model.dart';
import 'package:alem_live_mobile/features/rooms/data/rooms_api_client.dart';
import 'package:alem_live_mobile/features/rooms/domain/entities/recording_status.dart';
import 'package:alem_live_mobile/features/rooms/domain/entities/room_participant.dart';
import 'package:alem_live_mobile/features/rooms/domain/entities/room_session.dart';
import 'package:alem_live_mobile/features/rooms/domain/rooms_repository.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

final roomsRepositoryProvider = Provider<RoomsRepository>((ref) {
  return RoomsRepositoryImpl(
    apiClient: ref.watch(roomsApiClientProvider),
    config: ref.watch(appConfigProvider),
  );
});

class RoomsRepositoryImpl implements RoomsRepository {
  const RoomsRepositoryImpl({required this.apiClient, required this.config});

  final RoomsApiClient apiClient;
  final AppConfig config;

  @override
  Future<RoomSession> createRoom({
    required String roomName,
    required bool initialMicEnabled,
    required bool initialCameraEnabled,
  }) async {
    try {
      final response = await apiClient.create(
        roomName: roomName,
        initialMicEnabled: initialMicEnabled,
        initialCameraEnabled: initialCameraEnabled,
      );
      return _sessionFromResponse(response, roomName: roomName, isOwner: true);
    } catch (error) {
      if (!config.enableMockFallback) {
        throw mapDioException(error);
      }
      return _mockSession(roomName: roomName, isOwner: true);
    }
  }

  @override
  Future<RoomSession> joinRoom({required String roomName}) async {
    try {
      final response = await apiClient.join(roomName: roomName);
      return _sessionFromResponse(response, roomName: roomName, isOwner: false);
    } catch (error) {
      if (!config.enableMockFallback) {
        throw mapDioException(error);
      }
      return _mockSession(roomName: roomName, isOwner: false);
    }
  }

  @override
  Future<void> leaveRoom({required String roomId}) async {
    try {
      await apiClient.leave(roomId: roomId);
    } catch (error) {
      if (!config.enableMockFallback) {
        throw mapDioException(error);
      }
    }
  }

  @override
  Future<RoomSession?> leaveRoomWithInfo({required String roomId}) async {
    try {
      final response = await apiClient.leaveWithInfo(roomId: roomId);
      if (response.isEmpty) {
        return null;
      }
      return _sessionFromResponse(response, roomName: roomId, isOwner: false);
    } catch (error) {
      if (!config.enableMockFallback) {
        throw mapDioException(error);
      }
      return null;
    }
  }

  @override
  Future<RoomSession> roomInfo({required String roomId}) async {
    try {
      final response = await apiClient.roomInfo(roomId: roomId);
      return _sessionFromResponse(response, roomName: roomId, isOwner: false);
    } catch (error) {
      if (!config.enableMockFallback) {
        throw mapDioException(error);
      }
      return _mockSession(roomName: roomId, isOwner: false);
    }
  }

  @override
  Future<List<RoomParticipant>> participants({required String roomId}) async {
    try {
      final response = await apiClient.participants(roomId: roomId);
      return response
          .whereType<Map<String, dynamic>>()
          .map(RoomParticipantModel.fromJson)
          .toList();
    } catch (error) {
      if (!config.enableMockFallback) {
        throw mapDioException(error);
      }
      return const [];
    }
  }

  @override
  Future<RecordingStatus> startRecording({required String roomId}) async {
    try {
      return _recordingFromResponse(
        await apiClient.startRecording(roomId: roomId),
        roomId: roomId,
        fallbackState: RecordingState.recording,
      );
    } catch (error) {
      if (!config.enableMockFallback) {
        throw mapDioException(error);
      }
      return RecordingStatus(roomId: roomId, state: RecordingState.recording);
    }
  }

  @override
  Future<RecordingStatus> stopRecording({required String roomId}) async {
    try {
      return _recordingFromResponse(
        await apiClient.stopRecording(roomId: roomId),
        roomId: roomId,
        fallbackState: RecordingState.processing,
      );
    } catch (error) {
      if (!config.enableMockFallback) {
        throw mapDioException(error);
      }
      return RecordingStatus(roomId: roomId, state: RecordingState.processing);
    }
  }

  @override
  Future<RecordingStatus> recordingStatus({required String roomId}) async {
    try {
      return _recordingFromResponse(
        await apiClient.recordingStatus(roomId: roomId),
        roomId: roomId,
        fallbackState: RecordingState.idle,
      );
    } catch (error) {
      if (!config.enableMockFallback) {
        throw mapDioException(error);
      }
      return RecordingStatus(roomId: roomId, state: RecordingState.idle);
    }
  }

  @override
  Future<void> transferOwner({
    required String roomId,
    required String participantId,
  }) async {
    try {
      await apiClient.transferOwner(
        roomId: roomId,
        participantId: participantId,
      );
    } catch (error) {
      if (!config.enableMockFallback) {
        throw mapDioException(error);
      }
    }
  }

  @override
  Future<void> muteParticipant({
    required String roomId,
    required String participantId,
  }) {
    return _control(
      roomId: roomId,
      participantId: participantId,
      action: 'mute',
    );
  }

  @override
  Future<void> unmuteParticipant({
    required String roomId,
    required String participantId,
  }) {
    return _control(
      roomId: roomId,
      participantId: participantId,
      action: 'unmute',
    );
  }

  @override
  Future<void> cameraOff({
    required String roomId,
    required String participantId,
  }) {
    return _control(
      roomId: roomId,
      participantId: participantId,
      action: 'camera-off',
    );
  }

  @override
  Future<void> cameraOnRequest({
    required String roomId,
    required String participantId,
  }) {
    return _control(
      roomId: roomId,
      participantId: participantId,
      action: 'camera-on-request',
    );
  }

  @override
  Future<void> screenShareStarted({
    required String roomId,
    required String participantId,
  }) {
    return _control(
      roomId: roomId,
      participantId: participantId,
      action: 'screen-share-start',
    );
  }

  @override
  Future<void> screenShareStopped({
    required String roomId,
    required String participantId,
  }) {
    return _control(
      roomId: roomId,
      participantId: participantId,
      action: 'screen-share-stop',
    );
  }

  Future<void> _control({
    required String roomId,
    required String participantId,
    required String action,
  }) async {
    try {
      await apiClient.participantControl(
        roomId: roomId,
        participantId: participantId,
        action: action,
      );
    } catch (error) {
      if (!config.enableMockFallback) {
        throw mapDioException(error);
      }
    }
  }

  RoomSession _sessionFromResponse(
    Map<String, dynamic> response, {
    required String roomName,
    required bool isOwner,
  }) {
    final payload = _roomPayload(response);
    final liveKitToken =
        payload['liveKitToken']?.toString() ??
        payload['livekitToken']?.toString() ??
        payload['token']?.toString();
    final liveKitUrl =
        payload['liveKitUrl']?.toString() ??
        payload['livekitUrl']?.toString() ??
        payload['url']?.toString() ??
        (liveKitToken == null ? null : config.liveKitUrl);

    return RoomSession(
      roomId:
          payload['roomId']?.toString() ??
          payload['id']?.toString() ??
          _slug(roomName),
      roomName:
          payload['roomName']?.toString() ??
          payload['name']?.toString() ??
          roomName,
      isOwner: payload['isOwner'] as bool? ?? isOwner,
      ownerId: payload['ownerId']?.toString() ?? payload['owner']?.toString(),
      liveKitUrl: liveKitUrl,
      liveKitToken: liveKitToken,
    );
  }

  Map<String, dynamic> _roomPayload(Map<String, dynamic> response) {
    final room = response['room'];
    if (room is Map<String, dynamic>) {
      return {...room, ...response};
    }
    return response;
  }

  RoomSession _mockSession({required String roomName, required bool isOwner}) {
    return RoomSession(
      roomId: _slug(roomName),
      roomName: roomName,
      isOwner: isOwner,
    );
  }

  RecordingStatus _recordingFromResponse(
    Map<String, dynamic> response, {
    required String roomId,
    required RecordingState fallbackState,
  }) {
    final status =
        response['status']?.toString() ?? response['state']?.toString();
    final state = RecordingState.values.firstWhere(
      (value) => value.name == status,
      orElse: () => fallbackState,
    );
    return RecordingStatus(
      roomId: response['roomId']?.toString() ?? roomId,
      state: state,
      reportId: response['reportId']?.toString(),
    );
  }

  String _slug(String value) {
    return value
        .trim()
        .toLowerCase()
        .replaceAll(RegExp(r'[^a-z0-9а-яё]+'), '-')
        .replaceAll(RegExp('-+'), '-');
  }
}
