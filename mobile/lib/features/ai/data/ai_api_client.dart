import 'package:alem_live_mobile/core/network/dio_client.dart';
import 'package:alem_live_mobile/features/ai/data/models/ai_question_model.dart';
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
    String? roomName,
  }) async {
    final payload = AiQuestionModel(
      reportId: reportId,
      question: question,
      roomName: roomName,
    );

    try {
      final response = await _dio.post<Map<String, dynamic>>(
        '/api/reports/$reportId/ask-ai',
        data: payload.toAskAiJson(),
      );
      return response.data ?? <String, dynamic>{};
    } on DioException catch (error) {
      final statusCode = error.response?.statusCode;
      if (statusCode != 404 && statusCode != 405) {
        rethrow;
      }
    }

    final response = await _dio.post<Map<String, dynamic>>(
      '/api/reports/$reportId/chat',
      data: payload.toChatJson(),
    );
    return response.data ?? <String, dynamic>{};
  }
}
