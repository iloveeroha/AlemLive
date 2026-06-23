import 'package:alem_live_mobile/core/network/dio_client.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

final roomsApiClientProvider = Provider<RoomsApiClient>((ref) {
  return RoomsApiClient(ref.watch(dioProvider));
});

class RoomsApiClient {
  const RoomsApiClient(this._dio);

  final Dio _dio;

  Future<Map<String, dynamic>> create({
    required String roomName,
    required bool initialMicEnabled,
    required bool initialCameraEnabled,
  }) async {
    final response = await _dio.post<Map<String, dynamic>>(
      '/api/rooms/create',
      data: {
        'roomName': roomName,
        'initialMicEnabled': initialMicEnabled,
        'initialCameraEnabled': initialCameraEnabled,
      },
    );
    return response.data ?? <String, dynamic>{};
  }

  Future<Map<String, dynamic>> join({required String roomName}) async {
    final response = await _dio.post<Map<String, dynamic>>(
      '/api/rooms/join',
      data: {'roomName': roomName},
    );
    return response.data ?? <String, dynamic>{};
  }

  Future<void> leave({required String roomId}) async {
    await _dio.post<void>('/api/rooms/$roomId/leave');
  }

  Future<Map<String, dynamic>> leaveWithInfo({required String roomId}) async {
    final response = await _dio.post<Map<String, dynamic>>(
      '/api/rooms/$roomId/leave',
    );
    return response.data ?? <String, dynamic>{};
  }

  Future<Map<String, dynamic>> roomInfo({required String roomId}) async {
    final response = await _dio.get<Map<String, dynamic>>('/api/rooms/$roomId');
    return response.data ?? <String, dynamic>{};
  }

  Future<List<dynamic>> participants({required String roomId}) async {
    final response = await _dio.get<dynamic>('/api/rooms/$roomId/participants');
    final data = response.data;
    if (data is List<dynamic>) {
      return data;
    }
    if (data is Map<String, dynamic> && data['participants'] is List<dynamic>) {
      return data['participants'] as List<dynamic>;
    }
    return const [];
  }

  Future<Map<String, dynamic>> startRecording({required String roomId}) async {
    final response = await _dio.post<Map<String, dynamic>>(
      '/api/rooms/$roomId/recording/start',
    );
    return response.data ?? <String, dynamic>{};
  }

  Future<Map<String, dynamic>> stopRecording({required String roomId}) async {
    final response = await _dio.post<Map<String, dynamic>>(
      '/api/rooms/$roomId/recording/stop',
    );
    return response.data ?? <String, dynamic>{};
  }

  Future<Map<String, dynamic>> recordingStatus({required String roomId}) async {
    final response = await _dio.get<Map<String, dynamic>>(
      '/api/rooms/$roomId/recording/status',
    );
    return response.data ?? <String, dynamic>{};
  }

  Future<void> participantControl({
    required String roomId,
    required String participantId,
    required String action,
  }) async {
    await _dio.post<void>(
      '/api/rooms/$roomId/participants/$participantId/$action',
    );
  }
}
