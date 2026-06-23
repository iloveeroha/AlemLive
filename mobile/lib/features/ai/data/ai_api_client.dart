import 'package:alem_live_mobile/core/network/dio_client.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

final aiApiClientProvider = Provider<AiApiClient>((ref) {
  return AiApiClient(ref.watch(dioProvider));
});

class AiApiClient {
  const AiApiClient(this._dio);

  final Dio _dio;

  Future<Map<String, dynamic>> askQuestion({
    required String reportId,
    required String question,
  }) async {
    final response = await _dio.post<Map<String, dynamic>>(
      '/api/reports/$reportId/ask-ai',
      data: {'question': question},
    );
    return response.data ?? <String, dynamic>{};
  }
}
