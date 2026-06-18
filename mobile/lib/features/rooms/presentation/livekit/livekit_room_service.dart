import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:livekit_client/livekit_client.dart';

final liveKitRoomServiceProvider = Provider<LiveKitRoomService>((ref) {
  return const LiveKitRoomService();
});

class LiveKitRoomConnection {
  const LiveKitRoomConnection({required this.room, required this.listener});

  final Room room;
  final EventsListener<RoomEvent> listener;
}

class LiveKitRoomService {
  const LiveKitRoomService();

  Future<LiveKitRoomConnection> connect({
    required String url,
    required String token,
  }) async {
    final room = Room(
      roomOptions: const RoomOptions(adaptiveStream: true, dynacast: true),
    );
    final listener = room.createListener();

    try {
      await room.connect(url, token);
      return LiveKitRoomConnection(room: room, listener: listener);
    } catch (_) {
      await listener.dispose();
      await room.dispose();
      rethrow;
    }
  }

  Future<void> disconnect(LiveKitRoomConnection? connection) async {
    if (connection == null) {
      return;
    }
    await connection.listener.dispose();
    await connection.room.disconnect();
    await connection.room.dispose();
  }
}
