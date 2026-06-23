import 'dart:async';
import 'dart:convert';

import 'package:alem_live_mobile/features/auth/presentation/auth_controller.dart';
import 'package:alem_live_mobile/features/rooms/domain/entities/chat_message.dart';
import 'package:alem_live_mobile/features/rooms/domain/entities/recording_status.dart';
import 'package:alem_live_mobile/features/rooms/domain/entities/room_participant.dart';
import 'package:alem_live_mobile/features/rooms/domain/entities/room_session.dart';
import 'package:alem_live_mobile/features/rooms/domain/usecases/rooms_usecases.dart';
import 'package:alem_live_mobile/features/rooms/presentation/livekit/livekit_room_service.dart';
import 'package:alem_live_mobile/features/rooms/presentation/livekit/livekit_room_state.dart';
import 'package:alem_live_mobile/features/rooms/presentation/room_navigation_args.dart';
import 'package:alem_live_mobile/features/rooms/presentation/sync/room_sync_controller.dart';
import 'package:flutter/foundation.dart';
import 'package:livekit_client/livekit_client.dart';
import 'package:permission_handler/permission_handler.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

final liveKitRoomControllerProvider = Provider.autoDispose
    .family<LiveKitRoomController, RoomNavigationArgs>((ref, args) {
      final authState = ref.read(authControllerProvider);
      final user = authState.user;
      final currentUserId = user?.id;
      final currentUserName = user?.displayName ?? 'Вы';
      final controller = LiveKitRoomController(
        args: args,
        currentUserId: currentUserId,
        currentUserName: currentUserName,
        service: ref.watch(liveKitRoomServiceProvider),
        useCases: ref.watch(roomsUseCasesProvider),
        syncController: ref.watch(roomSyncControllerProvider(args)),
      );
      ref.onDispose(controller.dispose);
      controller.connect();
      return controller;
    });

class LiveKitRoomController extends ChangeNotifier {
  LiveKitRoomController({
    required this.args,
    required this.currentUserId,
    required this.currentUserName,
    required this.service,
    required this.useCases,
    required this.syncController,
  }) : state = LiveKitRoomState.initial(
         micEnabled: args.initialMicEnabled,
         cameraEnabled: args.initialCameraEnabled,
         ownerId: args.ownerId ?? (args.isOwner ? currentUserId : null),
         currentUserId: currentUserId,
       ) {
    syncController.addListener(_handleRoomSyncChanged);
  }

  static const _chatTopic = 'alemlive.chat';

  final RoomNavigationArgs args;
  final String? currentUserId;
  final String currentUserName;
  final LiveKitRoomService service;
  final RoomsUseCases useCases;
  final RoomSyncController syncController;

  LiveKitRoomState state;
  LiveKitRoomConnection? _connection;
  bool _disposed = false;

  Room? get _room => _connection?.room;
  LocalParticipant? get _localParticipant => _room?.localParticipant;
  bool get _preferBackendEventSync => true;

  Future<void> connect() async {
    if (!args.hasLiveKitCredentials) {
      _useMockFallback();
      return;
    }

    _setState(
      state.copyWith(
        status: LiveKitRoomStatus.connecting,
        clearErrorMessage: true,
      ),
    );

    try {
      await _ensureMediaPermissions();
      final connection = await service.connect(
        url: args.liveKitUrl!.trim(),
        token: args.liveKitToken!.trim(),
      );
      _connection = connection;
      _setUpListeners(connection);

      if (args.initialMicEnabled) {
        await connection.room.localParticipant?.setMicrophoneEnabled(true);
      }
      if (args.initialCameraEnabled) {
        await connection.room.localParticipant?.setCameraEnabled(true);
      }

      await _loadRecordingStatus();
      _syncParticipants(status: LiveKitRoomStatus.connected);
      unawaited(syncController.refreshRoomInfo());
    } catch (error) {
      _setState(
        state.copyWith(
          status: LiveKitRoomStatus.error,
          participants: const [],
          errorMessage: _connectionErrorMessage(error),
        ),
      );
    }
  }

  Future<void> toggleMicrophone() async {
    final nextValue = !state.micEnabled;

    if (state.isMock) {
      _updateMockCurrentUser(micEnabled: nextValue);
      return;
    }

    try {
      if (nextValue) {
        await _ensurePermission(Permission.microphone, 'микрофон');
      }
      await _localParticipant?.setMicrophoneEnabled(nextValue);
      _syncParticipants();
    } catch (error) {
      _showActionError(error);
    }
  }

  Future<void> toggleCamera() async {
    final nextValue = !state.cameraEnabled;

    if (state.isMock) {
      _updateMockCurrentUser(cameraEnabled: nextValue);
      return;
    }

    try {
      if (nextValue) {
        await _ensurePermission(Permission.camera, 'камеру');
      }
      await _localParticipant?.setCameraEnabled(nextValue);
      _syncParticipants();
    } catch (error) {
      _showActionError(error);
    }
  }

  Future<void> toggleScreenShare() async {
    final nextValue = !state.screenSharing;

    if (state.isMock) {
      _setState(
        state.copyWith(
          screenSharing: nextValue,
          roomNotice: nextValue ? 'Вы показываете экран' : null,
          clearRoomNotice: !nextValue,
        ),
      );
      return;
    }

    try {
      await _localParticipant?.setScreenShareEnabled(nextValue);
      _setState(
        state.copyWith(
          screenSharing: nextValue,
          roomNotice: nextValue ? 'Вы показываете экран' : null,
          clearRoomNotice: !nextValue,
        ),
      );
      _syncParticipants();
    } catch (error) {
      _setState(
        state.copyWith(
          controlErrorMessage:
              'Демонстрация экрана пока недоступна на этой платформе.',
        ),
      );
      // TODO: Add platform-specific screen sharing flow for iOS ReplayKit/web.
    }
  }

  Future<void> toggleRecording() async {
    if (_preferBackendEventSync) {
      await _toggleRecordingWithSync();
      return;
    }

    final nextValue = !state.isRecording;
    try {
      if (nextValue) {
        final status = await useCases.startRecording(roomId: args.roomId);
        _setState(
          state.copyWith(
            recordingState: status.state,
            roomNotice: 'Запись началась',
          ),
        );
      } else {
        final status = await useCases.stopRecording(roomId: args.roomId);
        _setState(
          state.copyWith(
            recordingState: status.state,
            roomNotice: 'Запись появится в отчетах после AI-обработки.',
          ),
        );
      }
    } catch (error) {
      _showActionError(error);
    }
  }

  Future<void> _toggleRecordingWithSync() async {
    final nextValue = !state.isRecording;
    try {
      if (nextValue) {
        final status = await useCases.startRecording(roomId: args.roomId);
        syncController.applyRecordingStatus(status, notice: 'Запись началась');
      } else {
        final status = await useCases.stopRecording(roomId: args.roomId);
        syncController.applyRecordingStatus(
          status,
          notice: 'Запись появится в отчетах после AI-обработки',
        );
      }
    } catch (error) {
      _showActionError(error);
    }
  }

  Future<void> sendChatMessage(String text) async {
    final trimmed = text.trim();
    if (trimmed.isEmpty) {
      return;
    }

    final message = ChatMessage(
      id: 'message-${DateTime.now().microsecondsSinceEpoch}',
      senderName: currentUserName,
      text: trimmed,
      sentAt: DateTime.now(),
      isMine: true,
    );
    _appendMessage(message);

    if (state.isMock || _localParticipant == null) {
      return;
    }

    try {
      final payload = jsonEncode({
        'id': message.id,
        'senderName': message.senderName,
        'text': message.text,
        'sentAt': message.sentAt.toIso8601String(),
      });
      await _localParticipant!.publishData(
        utf8.encode(payload),
        reliable: true,
        topic: _chatTopic,
      );
    } catch (error) {
      _showActionError(error);
    }
  }

  Future<void> toggleParticipantMicrophone(String participantId) async {
    final view = _participantById(participantId);
    if (view == null || !state.isCurrentUserOwner) {
      return;
    }

    final wasEnabled = view.participant.isMicEnabled;
    final key = ParticipantControlKey(
      participantId: participantId,
      type: ParticipantControlType.microphone,
    );
    await _runControl(key, () async {
      if (wasEnabled) {
        await useCases.muteParticipant(
          roomId: args.roomId,
          participantId: participantId,
        );
      } else {
        await useCases.unmuteParticipant(
          roomId: args.roomId,
          participantId: participantId,
        );
      }
      _setParticipantState(participantId, micEnabled: !wasEnabled);
      if (!syncController.state.isRealtimeConnected) {
        await syncController.refreshParticipants();
      }
    });
  }

  Future<void> toggleParticipantCamera(String participantId) async {
    final view = _participantById(participantId);
    if (view == null || !state.isCurrentUserOwner) {
      return;
    }

    final wasEnabled = view.participant.isCameraEnabled;
    final key = ParticipantControlKey(
      participantId: participantId,
      type: ParticipantControlType.camera,
    );
    await _runControl(key, () async {
      if (wasEnabled) {
        await useCases.cameraOff(
          roomId: args.roomId,
          participantId: participantId,
        );
      } else {
        await useCases.cameraOnRequest(
          roomId: args.roomId,
          participantId: participantId,
        );
      }
      _setParticipantState(participantId, cameraEnabled: !wasEnabled);
      if (!syncController.state.isRealtimeConnected) {
        await syncController.refreshParticipants();
      }
    });
  }

  Future<RoomSession?> leaveRoom() async {
    final updatedRoom = await useCases.leaveRoomWithInfo(roomId: args.roomId);
    await syncController.disconnect();
    await disconnect();
    return updatedRoom;
  }

  Future<void> disconnect() async {
    final connection = _connection;
    _connection = null;
    await service.disconnect(connection);
    if (!_disposed) {
      _setState(state.copyWith(status: LiveKitRoomStatus.disconnected));
    }
  }

  Future<void> syncRoomInfo() async {
    if (_preferBackendEventSync) {
      await syncController.refreshRoomInfo();
      return;
    }

    try {
      // TODO: Replace polling/manual refresh with backend room events/WebSocket.
      final info = await useCases.roomInfo(roomId: args.roomId);
      _applyRoomInfo(info);
    } catch (error) {
      if (!state.isMock) {
        _showActionError(error);
      }
    }
  }

  void _handleRoomSyncChanged() {
    final syncState = syncController.state;
    if (syncState.roomClosed) {
      unawaited(disconnect());
    }

    _setState(
      state.copyWith(
        ownerId: syncState.ownerId,
        recordingState: syncState.recordingState,
        roomNotice: syncState.notice,
        controlErrorMessage: syncState.errorMessage,
      ),
    );

    if (_room != null) {
      _syncParticipants();
    } else if (syncState.participants.isNotEmpty) {
      _mergeBackendParticipantsIntoState();
    }
  }

  Future<void> _loadRecordingStatus() async {
    try {
      final status = await useCases.recordingStatus(roomId: args.roomId);
      _setState(state.copyWith(recordingState: status.state));
    } catch (_) {
      // Mock fallback in repository already keeps the app usable.
    }
  }

  Future<void> _runControl(
    ParticipantControlKey key,
    Future<void> Function() action,
  ) async {
    _setState(
      state.copyWith(
        controlLoading: {...state.controlLoading, key},
        clearControlErrorMessage: true,
      ),
    );
    try {
      await action();
    } catch (error) {
      _setState(state.copyWith(controlErrorMessage: error.toString()));
    } finally {
      final nextLoading = {...state.controlLoading}..remove(key);
      _setState(state.copyWith(controlLoading: nextLoading));
    }
  }

  void _setUpListeners(LiveKitRoomConnection connection) {
    connection.room.addListener(_handleRoomChanged);
    connection.listener
      ..on<ParticipantEvent>((_) => _syncParticipants())
      ..on<ParticipantDisconnectedEvent>((event) {
        if (event.participant.identity == state.ownerId) {
          unawaited(syncRoomInfo());
        }
        _syncParticipants();
      })
      ..on<DataReceivedEvent>(_handleDataMessage)
      ..on<RoomReconnectingEvent>((_) {
        _setState(state.copyWith(status: LiveKitRoomStatus.reconnecting));
      })
      ..on<RoomReconnectedEvent>((_) {
        _syncParticipants(status: LiveKitRoomStatus.connected);
        unawaited(syncRoomInfo());
      })
      ..on<RoomDisconnectedEvent>((_) {
        _setState(state.copyWith(status: LiveKitRoomStatus.disconnected));
      });
  }

  void _handleRoomChanged() {
    _syncParticipants();
  }

  void _handleDataMessage(DataReceivedEvent event) {
    if (event.topic != _chatTopic) {
      return;
    }
    try {
      final decoded = jsonDecode(utf8.decode(event.data));
      if (decoded is! Map<String, dynamic>) {
        return;
      }
      final senderName =
          decoded['senderName']?.toString() ??
          event.participant?.name ??
          event.participant?.identity ??
          'Участник';
      _appendMessage(
        ChatMessage(
          id:
              decoded['id']?.toString() ??
              'message-${DateTime.now().microsecondsSinceEpoch}',
          senderName: senderName,
          text: decoded['text']?.toString() ?? '',
          sentAt:
              DateTime.tryParse(decoded['sentAt']?.toString() ?? '') ??
              DateTime.now(),
          isMine: false,
        ),
      );
    } catch (error) {
      _showActionError(error);
    }
  }

  void _appendMessage(ChatMessage message) {
    _setState(state.copyWith(messages: [...state.messages, message]));
  }

  void _syncParticipants({LiveKitRoomStatus? status}) {
    final room = _room;
    if (room == null) {
      return;
    }

    final views = <RoomParticipantView>[];
    final localParticipant = room.localParticipant;
    if (localParticipant != null) {
      views.add(_participantView(participant: localParticipant));
    }

    for (final participant in room.remoteParticipants.values) {
      views.add(_participantView(participant: participant));
    }
    final mergedViews = _mergeBackendParticipants(views);

    _setState(
      state.copyWith(
        status: status ?? state.status,
        participants: mergedViews,
        micEnabled: localParticipant?.isMicrophoneEnabled() ?? state.micEnabled,
        cameraEnabled:
            localParticipant?.isCameraEnabled() ?? state.cameraEnabled,
        screenSharing:
            localParticipant?.isScreenShareEnabled() ?? state.screenSharing,
        ownerId: syncController.state.ownerId,
        recordingState:
            syncController.state.recordingState ?? state.recordingState,
        roomNotice: syncController.state.notice,
        controlErrorMessage: syncController.state.errorMessage,
        clearErrorMessage: true,
      ),
    );
  }

  List<RoomParticipantView> _mergeBackendParticipants(
    List<RoomParticipantView> liveKitViews,
  ) {
    final syncState = syncController.state;
    final ownerId = syncState.ownerId ?? state.ownerId;
    final result = <RoomParticipantView>[];
    final seenIds = <String>{};

    for (final view in liveKitViews) {
      final backend = syncState.participants[view.participant.id];
      seenIds.add(view.participant.id);
      result.add(
        RoomParticipantView(
          participant: _mergeParticipant(
            view.participant,
            backend,
            ownerId: ownerId,
          ),
          videoTrack: view.videoTrack,
        ),
      );
    }

    for (final backend in syncState.participants.values) {
      if (seenIds.contains(backend.id)) {
        continue;
      }
      result.add(
        RoomParticipantView(
          participant: _mergeParticipant(backend, null, ownerId: ownerId),
        ),
      );
    }

    return result;
  }

  void _mergeBackendParticipantsIntoState() {
    final mergedViews = _mergeBackendParticipants(state.participants);
    _setState(state.copyWith(participants: mergedViews));
  }

  RoomParticipant _mergeParticipant(
    RoomParticipant base,
    RoomParticipant? backend, {
    required String? ownerId,
  }) {
    return RoomParticipant(
      id: base.id,
      name: backend?.name ?? base.name,
      isCurrentUser: base.isCurrentUser,
      isOwner:
          base.id == ownerId ||
          (base.isCurrentUser && ownerId == currentUserId),
      isMicEnabled: backend?.isMicEnabled ?? base.isMicEnabled,
      isCameraEnabled: backend?.isCameraEnabled ?? base.isCameraEnabled,
    );
  }

  RoomParticipantView _participantView({required Participant participant}) {
    final participantId = participant.identity;
    final isCurrentUser = participant is LocalParticipant;
    final isOwner =
        state.ownerId == participantId ||
        (isCurrentUser && state.ownerId == currentUserId) ||
        (isCurrentUser && args.isOwner && state.ownerId == null);
    final name = participant.name.trim().isNotEmpty
        ? participant.name.trim()
        : participant.identity;
    final cameraPublication = participant.getTrackPublicationBySource(
      TrackSource.camera,
    );
    final cameraTrack = cameraPublication?.track;
    final hasCamera = participant.isCameraEnabled();

    return RoomParticipantView(
      participant: RoomParticipant(
        id: participantId,
        name: isCurrentUser ? currentUserName : name,
        isCurrentUser: isCurrentUser,
        isOwner: isOwner,
        isMicEnabled: participant.isMicrophoneEnabled(),
        isCameraEnabled: hasCamera,
      ),
      videoTrack: hasCamera && cameraTrack is VideoTrack ? cameraTrack : null,
    );
  }

  void _useMockFallback() {
    final ownerId = state.ownerId ?? (args.isOwner ? 'current-user' : 'owner');
    _setState(
      LiveKitRoomState(
        status: LiveKitRoomStatus.mock,
        micEnabled: args.initialMicEnabled,
        cameraEnabled: args.initialCameraEnabled,
        screenSharing: false,
        recordingState: RecordingState.idle,
        ownerId: ownerId,
        currentUserId: 'current-user',
        controlLoading: const <ParticipantControlKey>{},
        messages: [
          ChatMessage(
            id: 'message-1',
            senderName: 'Алия Нурлан',
            text: 'Всем привет, я уже в комнате.',
            sentAt: DateTime.now().subtract(const Duration(minutes: 3)),
            isMine: false,
          ),
          ChatMessage(
            id: 'message-2',
            senderName: currentUserName,
            text: 'Проверяю микрофон и камеру.',
            sentAt: DateTime.now().subtract(const Duration(minutes: 2)),
            isMine: true,
          ),
        ],
        participants: [
          RoomParticipantView(
            participant: RoomParticipant(
              id: 'current-user',
              name: currentUserName,
              isCurrentUser: true,
              isOwner: ownerId == 'current-user',
              isMicEnabled: args.initialMicEnabled,
              isCameraEnabled: args.initialCameraEnabled,
            ),
          ),
          RoomParticipantView(
            participant: RoomParticipant(
              id: 'owner',
              name: 'Алия Нурлан',
              isCurrentUser: false,
              isOwner: ownerId == 'owner',
              isMicEnabled: true,
              isCameraEnabled: true,
            ),
          ),
          const RoomParticipantView(
            participant: RoomParticipant(
              id: 'participant-2',
              name: 'Данияр Сейтов',
              isCurrentUser: false,
              isOwner: false,
              isMicEnabled: false,
              isCameraEnabled: true,
            ),
          ),
          const RoomParticipantView(
            participant: RoomParticipant(
              id: 'participant-3',
              name: 'Madi QA',
              isCurrentUser: false,
              isOwner: false,
              isMicEnabled: true,
              isCameraEnabled: false,
            ),
          ),
        ],
      ),
    );
  }

  void _updateMockCurrentUser({bool? micEnabled, bool? cameraEnabled}) {
    final nextMic = micEnabled ?? state.micEnabled;
    final nextCamera = cameraEnabled ?? state.cameraEnabled;
    _setState(
      state.copyWith(
        micEnabled: nextMic,
        cameraEnabled: nextCamera,
        participants: state.participants.map((view) {
          if (!view.participant.isCurrentUser) {
            return view;
          }
          return RoomParticipantView(
            participant: view.participant.copyWith(
              isMicEnabled: nextMic,
              isCameraEnabled: nextCamera,
            ),
            videoTrack: view.videoTrack,
          );
        }).toList(),
      ),
    );
  }

  void _setParticipantState(
    String participantId, {
    bool? micEnabled,
    bool? cameraEnabled,
  }) {
    _setState(
      state.copyWith(
        participants: state.participants.map((view) {
          if (view.participant.id != participantId) {
            return view;
          }
          return RoomParticipantView(
            participant: view.participant.copyWith(
              isMicEnabled: micEnabled,
              isCameraEnabled: cameraEnabled,
            ),
            videoTrack: view.videoTrack,
          );
        }).toList(),
      ),
    );
  }

  RoomParticipantView? _participantById(String participantId) {
    for (final view in state.participants) {
      if (view.participant.id == participantId) {
        return view;
      }
    }
    return null;
  }

  void _applyRoomInfo(RoomSession session) {
    final ownerId = session.ownerId ?? state.ownerId;
    _setState(
      state.copyWith(
        ownerId: ownerId,
        participants: state.participants.map((view) {
          return RoomParticipantView(
            participant: view.participant.copyWith(
              isOwner:
                  view.participant.id == ownerId ||
                  (view.participant.isCurrentUser && ownerId == currentUserId),
            ),
            videoTrack: view.videoTrack,
          );
        }).toList(),
      ),
    );
  }

  Future<void> _ensureMediaPermissions() async {
    if (args.initialMicEnabled) {
      await _ensurePermission(Permission.microphone, 'микрофон');
    }
    if (args.initialCameraEnabled) {
      await _ensurePermission(Permission.camera, 'камеру');
    }
  }

  Future<void> _ensurePermission(Permission permission, String label) async {
    final status = await permission.request();
    if (!status.isGranted) {
      throw StateError('Разрешите доступ к $label, чтобы войти в комнату.');
    }
  }

  void _showActionError(Object error) {
    _setState(
      state.copyWith(controlErrorMessage: _connectionErrorMessage(error)),
    );
  }

  String _connectionErrorMessage(Object error) {
    if (error is StateError) {
      return error.message;
    }
    return 'Не удалось выполнить действие: $error';
  }

  void _setState(LiveKitRoomState nextState) {
    if (_disposed) {
      return;
    }
    state = nextState;
    notifyListeners();
  }

  @override
  void dispose() {
    _disposed = true;
    syncController.removeListener(_handleRoomSyncChanged);
    _room?.removeListener(_handleRoomChanged);
    unawaited(service.disconnect(_connection));
    super.dispose();
  }
}
