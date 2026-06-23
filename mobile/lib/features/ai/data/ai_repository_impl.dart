import 'package:alem_live_mobile/app/config.dart';
import 'package:alem_live_mobile/core/network/dio_client.dart';
import 'package:alem_live_mobile/features/ai/data/ai_api_client.dart';
import 'package:alem_live_mobile/features/ai/data/models/ai_answer_model.dart';
import 'package:alem_live_mobile/features/ai/domain/ai_repository.dart';
import 'package:alem_live_mobile/features/ai/domain/entities/ai_answer.dart';
import 'package:alem_live_mobile/features/ai/domain/entities/ai_question.dart';
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
  Future<AiAnswer> askForAnswer({
    required AiQuestion question,
    required String fallbackTakeaway,
  }) async {
    try {
      final response = await apiClient.askQuestion(
        reportId: question.reportId,
        question: question.question,
        roomName: question.roomName,
      );
      return AiAnswerModel.fromJson(
        reportId: question.reportId,
        question: question.question,
        json: response,
      );
    } catch (error) {
      if (!config.enableMockFallback) {
        throw mapDioException(error);
      }
      return AiAnswerModel(
        reportId: question.reportId,
        question: question.question,
        answer:
            'Mock AI: по встрече "${question.roomName ?? 'AlemLive'}" главный вывод такой: $fallbackTakeaway',
        language: 'ru',
      );
    }
  }

  @override
  Future<String> askQuestion({
    required String reportId,
    required String roomName,
    required String question,
    required String fallbackTakeaway,
  }) async {
    final answer = await askForAnswer(
      question: AiQuestion(
        reportId: reportId,
        question: question,
        roomName: roomName,
      ),
      fallbackTakeaway: fallbackTakeaway,
    );
    return answer.answer;
  }
}
