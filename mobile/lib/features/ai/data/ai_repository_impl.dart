import 'package:alem_live_mobile/app/config.dart';
import 'package:alem_live_mobile/core/network/dio_client.dart';
import 'package:alem_live_mobile/features/ai/data/ai_api_client.dart';
import 'package:alem_live_mobile/features/ai/domain/ai_repository.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

final aiRepositoryProvider = Provider<AiRepository>((ref) {
  return AiRepositoryImpl(
    apiClient: ref.watch(aiApiClientProvider),
    config: ref.watch(appConfigProvider),
  );
});

class AiRepositoryImpl implements AiRepository {
  const AiRepositoryImpl({required this.apiClient, required this.config});

  final AiApiClient apiClient;
  final AppConfig config;

  @override
  Future<String> askQuestion({
    required String reportId,
    required String roomName,
    required String question,
    required String fallbackTakeaway,
  }) async {
    try {
      final response = await apiClient.askQuestion(
        reportId: reportId,
        question: question,
      );
      return (response['answer'] ?? response['text'] ?? '').toString();
    } catch (error) {
      if (!config.enableMockFallback) {
        throw mapDioException(error);
      }
      return 'Mock AI: по встрече "$roomName" главный вывод такой: $fallbackTakeaway';
    }
  }
}
