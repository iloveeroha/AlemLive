import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:alem_live_mobile/app/config.dart';
import 'package:alem_live_mobile/core/network/dio_client.dart';
import 'package:alem_live_mobile/core/storage/secure_storage_service.dart';
import 'package:alem_live_mobile/features/rooms/presentation/sync/room_sync_state.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

final roomEventsServiceProvider = Provider<RoomEventsService>((ref) {
  return RoomEventsService(
    config: ref.watch(appConfigProvider),
    storage: ref.watch(secureStorageServiceProvider),
  );
});

class RoomEventsConnection {
  RoomEventsConnection({
    required String roomId,
    required AppConfig config,
    required SecureStorageService storage,
  }) : this._(roomId, config, storage);

  RoomEventsConnection._(this._roomId, this._config, this._storage);

  final String _roomId;
  final AppConfig _config;
  final SecureStorageService _storage;
  final _eventsController = StreamController<RoomEventMessage>.broadcast();
  final _statusController =
      StreamController<RoomEventsConnectionStatus>.broadcast();

  WebSocket? _socket;
  bool _closed = false;
  int _attempt = 0;

  Stream<RoomEventMessage> get events => _eventsController.stream;
  Stream<RoomEventsConnectionStatus> get statuses => _statusController.stream;

  Future<void> start() async {
    unawaited(_connectLoop());
  }

  Future<void> disconnect() async {
    _closed = true;
    _statusController.add(RoomEventsConnectionStatus.disconnected);
    await _socket?.close();
    await _eventsController.close();
    await _statusController.close();
  }

  Future<void> _connectLoop() async {
    while (!_closed) {
      _statusController.add(
        _attempt == 0
            ? RoomEventsConnectionStatus.connecting
            : RoomEventsConnectionStatus.reconnecting,
      );

      try {
        final token = await _storage.readAccessToken();
        final uri = _eventsUri(token);
        _socket = await WebSocket.connect(
          uri.toString(),
          headers: {
            if (token != null && token.isNotEmpty)
              HttpHeaders.authorizationHeader: 'Bearer $token',
          },
        );
        _attempt = 0;
        _statusController.add(RoomEventsConnectionStatus.connected);

        await for (final message in _socket!) {
          if (_closed) {
            return;
          }
          final event = _parseEvent(message);
          if (event != null) {
            _eventsController.add(event);
          }
        }
      } catch (error) {
        if (_closed) {
          return;
        }
        _statusController.add(RoomEventsConnectionStatus.error);
      }

      if (!_closed) {
        _attempt += 1;
        final delaySeconds = (_attempt + 1).clamp(2, 12);
        await Future<void>.delayed(Duration(seconds: delaySeconds));
      }
    }
  }

  Uri _eventsUri(String? token) {
    final base = Uri.parse(_config.backendBaseUrl);
    final scheme = base.scheme == 'https' ? 'wss' : 'ws';
    final basePath = base.path.endsWith('/')
        ? base.path.substring(0, base.path.length - 1)
        : base.path;
    final path = '$basePath/api/rooms/$_roomId/events';

    return base.replace(
      scheme: scheme,
      path: path,
      queryParameters: {
        ...base.queryParameters,
        if (token != null && token.isNotEmpty) 'token': token,
      },
    );
  }

  RoomEventMessage? _parseEvent(dynamic message) {
    try {
      final decoded = jsonDecode(message.toString());
      if (decoded is Map<String, dynamic>) {
        return RoomEventMessage.fromJson(decoded);
      }
    } catch (_) {
      return null;
    }
    return null;
  }
}

class RoomEventsService {
  const RoomEventsService({required this.config, required this.storage});

  final AppConfig config;
  final SecureStorageService storage;

  RoomEventsConnection connect({required String roomId}) {
    return RoomEventsConnection(
      roomId: roomId,
      config: config,
      storage: storage,
    )..start();
  }
}
