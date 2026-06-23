import 'dart:async';

import 'package:alem_live_mobile/app/config.dart';
import 'package:alem_live_mobile/core/network/dio_client.dart';
import 'package:alem_live_mobile/features/auth/presentation/auth_controller.dart';
import 'package:alem_live_mobile/features/rooms/domain/entities/recording_status.dart';
import 'package:alem_live_mobile/features/rooms/domain/entities/room_participant.dart';
import 'package:alem_live_mobile/features/rooms/domain/entities/room_session.dart';
import 'package:alem_live_mobile/features/rooms/domain/usecases/rooms_usecases.dart';
import 'package:alem_live_mobile/features/rooms/presentation/room_navigation_args.dart';
import 'package:alem_live_mobile/features/rooms/presentation/sync/room_events_service.dart';
import 'package:alem_live_mobile/features/rooms/presentation/sync/room_sync_state.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

final roomSyncControllerProvider = Provider.autoDispose
    .family<RoomSyncController, RoomNavigationArgs>((ref, args) {
      final authState = ref.read(authControllerProvider);
      final controller = RoomSyncController(
        args: args,
        currentUserId: authState.user?.id,
        currentUserName: authState.user?.displayName ?? 'Вы',
        config: ref.watch(appConfigProvider),
        eventsService: ref.watch(roomEventsServiceProvider),
        useCases: ref.watch(roomsUseCasesProvider),
      );
      ref.onDispose(controller.dispose);
      controller.start();
      return controller;
    });

class RoomSyncController extends ChangeNotifier {
  RoomSyncController({
    required this.args,
    required this.currentUserId,
    required this.currentUserName,
    required this.config,
    required this.eventsService,
    required this.useCases,
  }) : state = const RoomSyncState.initial();

  final RoomNavigationArgs args;
  final String? currentUserId;
  final String currentUserName;
  final AppConfig config;
  final RoomEventsService eventsService;
  final RoomsUseCases useCases;

  RoomSyncState state;
  RoomEventsConnection? _connection;
  StreamSubscription<RoomEventMessage>? _eventSubscription;
  StreamSubscription<RoomEventsConnectionStatus>? _statusSubscription;
  Timer? _pollingTimer;
  Timer? _mockTimer;
  bool _disposed = false;
  int _mockStep = 0;

  Future<void> start() async {
    await Future.wait([
      refreshRoomInfo(),
      refreshParticipants(),
      refreshRecordingStatus(),
    ]);

    if (args.hasLiveKitCredentials) {
      _connectEvents();
    } else if (config.enableMockFallback) {
      _startMockEvents();
    } else {
      _startPollingFallback();
    }
  }

  Future<void> disconnect() async {
    await _eventSubscription?.cancel();
    await _statusSubscription?.cancel();
    await _connection?.disconnect();
    _pollingTimer?.cancel();
    _mockTimer?.cancel();
    _setState(
      state.copyWith(connectionStatus: RoomEventsConnectionStatus.disconnected),
    );
  }

  Future<void> refreshRoomInfo() async {
    try {
      final info = await useCases.roomInfo(roomId: args.roomId);
      _applyRoomInfo(info);
    } catch (error) {
      if (!config.enableMockFallback) {
        _setError(error);
      }
    }
  }

  Future<void> refreshParticipants() async {
    try {
      final participants = await useCases.participants(roomId: args.roomId);
      if (participants.isEmpty) {
        return;
      }
      final nextParticipants = <String, RoomParticipant>{
        for (final participant in participants) participant.id: participant,
      };
      _setState(state.copyWith(participants: nextParticipants));
    } catch (error) {
      if (!config.enableMockFallback) {
        _setError(error);
      }
    }
  }

  Future<void> refreshRecordingStatus() async {
    try {
      final status = await useCases.recordingStatus(roomId: args.roomId);
      _setRecordingStatus(status.state);
    } catch (error) {
      if (!config.enableMockFallback) {
        _setError(error);
      }
    }
  }

  void applyRecordingStatus(RecordingStatus status, {String? notice}) {
    _setRecordingStatus(status.state, notice: notice);
  }

  void _connectEvents() {
    _connection = eventsService.connect(roomId: args.roomId);
    _eventSubscription = _connection!.events.listen(
      _handleEvent,
      onError: _setError,
    );
    _statusSubscription = _connection!.statuses.listen(_handleStatus);
  }

  void _handleStatus(RoomEventsConnectionStatus status) {
    _setState(state.copyWith(connectionStatus: status));
    if (status == RoomEventsConnectionStatus.connected) {
      _pollingTimer?.cancel();
      _pollingTimer = null;
      return;
    }
    if (status == RoomEventsConnectionStatus.error ||
        status == RoomEventsConnectionStatus.reconnecting) {
      _startPollingFallback();
    }
  }

  void _startPollingFallback() {
    if (_pollingTimer != null) {
      return;
    }
    _setState(
      state.copyWith(connectionStatus: RoomEventsConnectionStatus.polling),
    );
    _pollingTimer = Timer.periodic(const Duration(seconds: 5), (_) {
      unawaited(refreshRoomInfo());
      unawaited(refreshParticipants());
      unawaited(refreshRecordingStatus());
    });
  }

  void _startMockEvents() {
    _setState(
      state.copyWith(
        connectionStatus: RoomEventsConnectionStatus.mock,
        ownerId: args.ownerId ?? (args.isOwner ? 'current-user' : 'owner'),
      ),
    );
    _mockTimer = Timer.periodic(const Duration(seconds: 4), (_) {
      _mockStep += 1;
      switch (_mockStep) {
        case 1:
          _handleEvent(
            const RoomEventMessage(
              type: RoomEventType.participantJoined,
              payload: {
                'participantId': 'mock-sync-user',
                'name': 'Mock Sync',
                'micEnabled': true,
                'cameraEnabled': false,
              },
            ),
          );
        case 2:
          _handleEvent(
            const RoomEventMessage(
              type: RoomEventType.participantMicChanged,
              payload: {'participantId': 'mock-sync-user', 'micEnabled': false},
            ),
          );
        case 3:
          _handleEvent(
            const RoomEventMessage(
              type: RoomEventType.ownerChanged,
              payload: {'ownerId': 'current-user'},
            ),
          );
        case 4:
          _handleEvent(
            const RoomEventMessage(
              type: RoomEventType.recordingStarted,
              payload: {},
            ),
          );
        case 5:
          _handleEvent(
            const RoomEventMessage(
              type: RoomEventType.recordingStopped,
              payload: {},
            ),
          );
          _mockTimer?.cancel();
      }
    });
  }

  void _handleEvent(RoomEventMessage event) {
    switch (event.type) {
      case RoomEventType.participantJoined:
        _upsertParticipant(_participantFromPayload(event.payload));
      case RoomEventType.participantLeft:
        _removeParticipant(_participantId(event.payload));
      case RoomEventType.participantMicChanged:
        _updateParticipantMedia(
          _participantId(event.payload),
          micEnabled:
              _boolValue(event.payload, 'micEnabled') ??
              _boolValue(event.payload, 'isMicEnabled') ??
              _boolValue(event.payload, 'enabled'),
        );
      case RoomEventType.participantCameraChanged:
        _updateParticipantMedia(
          _participantId(event.payload),
          cameraEnabled:
              _boolValue(event.payload, 'cameraEnabled') ??
              _boolValue(event.payload, 'isCameraEnabled') ??
              _boolValue(event.payload, 'enabled'),
        );
      case RoomEventType.participantScreenShareChanged:
        _updateParticipantMedia(
          _participantId(event.payload),
          screenSharing:
              _boolValue(event.payload, 'screenSharing') ??
              _boolValue(event.payload, 'isScreenSharing') ??
              _boolValue(event.payload, 'enabled'),
        );
      case RoomEventType.ownerChanged:
        _applyOwnerChanged(event.payload);
      case RoomEventType.recordingStarted:
        _setRecordingStatus(
          RecordingState.recording,
          notice: 'Запись началась',
        );
      case RoomEventType.recordingStopped:
        _setRecordingStatus(
          RecordingState.processing,
          notice: 'Запись появится в отчетах после AI-обработки',
        );
      case RoomEventType.recordingStatusChanged:
        _setRecordingStatus(_recordingStateFromPayload(event.payload));
      case RoomEventType.roomClosed:
        _setState(
          state.copyWith(
            roomClosed: true,
            notice: 'Комната закрыта',
            connectionStatus: RoomEventsConnectionStatus.disconnected,
          ),
        );
      // TODO: Add push notifications for users who are outside the room.
      case RoomEventType.unknown:
        break;
    }
  }

  void _applyRoomInfo(RoomSession info) {
    _setState(state.copyWith(ownerId: info.ownerId, clearErrorMessage: true));
  }

  void _applyOwnerChanged(Map<String, dynamic> payload) {
    final ownerId =
        payload['ownerId']?.toString() ??
        payload['newOwnerId']?.toString() ??
        payload['participantId']?.toString();
    if (ownerId == null || ownerId.isEmpty) {
      return;
    }
    _setState(
      state.copyWith(
        ownerId: ownerId,
        notice: ownerId == currentUserId ? 'Вы стали создателем' : null,
        clearNotice: ownerId != currentUserId,
      ),
    );
  }

  void _upsertParticipant(RoomParticipant participant) {
    _setState(
      state.copyWith(
        participants: {...state.participants, participant.id: participant},
      ),
    );
  }

  void _removeParticipant(String? participantId) {
    if (participantId == null || participantId.isEmpty) {
      return;
    }
    final participants = {...state.participants}..remove(participantId);
    _setState(state.copyWith(participants: participants));
  }

  void _updateParticipantMedia(
    String? participantId, {
    bool? micEnabled,
    bool? cameraEnabled,
    bool? screenSharing,
  }) {
    if (participantId == null || participantId.isEmpty) {
      return;
    }
    final existing = state.participants[participantId];
    final participant =
        existing ??
        RoomParticipant(
          id: participantId,
          name: participantId,
          isCurrentUser: participantId == currentUserId,
          isOwner: participantId == state.ownerId,
          isMicEnabled: micEnabled ?? true,
          isCameraEnabled: cameraEnabled ?? true,
          isScreenSharing: screenSharing ?? false,
        );
    _upsertParticipant(
      participant.copyWith(
        isMicEnabled: micEnabled,
        isCameraEnabled: cameraEnabled,
        isScreenSharing: screenSharing,
        isOwner: participantId == state.ownerId,
      ),
    );
  }

  void _setRecordingStatus(RecordingState status, {String? notice}) {
    _setState(
      state.copyWith(
        recordingState: status,
        notice: notice,
        clearNotice: notice == null,
      ),
    );
  }

  RecordingState _recordingStateFromPayload(Map<String, dynamic> payload) {
    final status =
        payload['status']?.toString() ?? payload['state']?.toString();
    return RecordingState.values.firstWhere(
      (value) => value.name == status,
      orElse: () => state.recordingState ?? RecordingState.idle,
    );
  }

  RoomParticipant _participantFromPayload(Map<String, dynamic> payload) {
    final nested = payload['participant'];
    final data = nested is Map<String, dynamic> ? nested : payload;
    final participantId =
        _participantId(data) ?? 'participant-${data.hashCode}';
    return RoomParticipant(
      id: participantId,
      name:
          data['name']?.toString() ??
          data['displayName']?.toString() ??
          data['username']?.toString() ??
          participantId,
      isCurrentUser: participantId == currentUserId,
      isOwner: participantId == state.ownerId,
      isMicEnabled:
          _boolValue(data, 'micEnabled') ??
          _boolValue(data, 'isMicEnabled') ??
          true,
      isCameraEnabled:
          _boolValue(data, 'cameraEnabled') ??
          _boolValue(data, 'isCameraEnabled') ??
          true,
      isScreenSharing:
          _boolValue(data, 'screenSharing') ??
          _boolValue(data, 'isScreenSharing') ??
          false,
    );
  }

  String? _participantId(Map<String, dynamic> payload) {
    return payload['participantId']?.toString() ??
        payload['id']?.toString() ??
        payload['userId']?.toString() ??
        payload['identity']?.toString();
  }

  bool? _boolValue(Map<String, dynamic> payload, String key) {
    final value = payload[key];
    if (value is bool) {
      return value;
    }
    if (value is String) {
      return value.toLowerCase() == 'true';
    }
    return null;
  }

  void _setError(Object error) {
    _setState(state.copyWith(errorMessage: error.toString()));
  }

  void _setState(RoomSyncState nextState) {
    if (_disposed) {
      return;
    }
    state = nextState;
    notifyListeners();
  }

  @override
  void dispose() {
    _disposed = true;
    _pollingTimer?.cancel();
    _mockTimer?.cancel();
    unawaited(disconnect());
    super.dispose();
  }
}
