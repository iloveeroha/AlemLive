import 'package:alem_live_mobile/features/ai/data/ai_repository_impl.dart';
import 'package:alem_live_mobile/features/ai/domain/ai_repository.dart';
import 'package:alem_live_mobile/features/ai/domain/entities/ai_answer.dart';
import 'package:alem_live_mobile/features/ai/domain/entities/ai_question.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

final askAiQuestionUseCaseProvider = Provider<AskAiQuestionUseCase>((ref) {
  return AskAiQuestionUseCase(ref.watch(aiRepositoryProvider));
});

final askAiAnswerUseCaseProvider = Provider<AskAiAnswerUseCase>((ref) {
  return AskAiAnswerUseCase(ref.watch(aiRepositoryProvider));
});

class AskAiQuestionUseCase {
  const AskAiQuestionUseCase(this._repository);

  final AiRepository _repository;

  Future<String> call({
    required String reportId,
    required String roomName,
    required String question,
    required String fallbackTakeaway,
  }) {
    return _repository.askQuestion(
      reportId: reportId,
      roomName: roomName,
      question: question,
      fallbackTakeaway: fallbackTakeaway,
    );
  }
}

class AskAiAnswerUseCase {
  const AskAiAnswerUseCase(this._repository);

  final AiRepository _repository;

  Future<AiAnswer> call({
    required AiQuestion question,
    required String fallbackTakeaway,
  }) {
    return _repository.askForAnswer(
      question: question,
      fallbackTakeaway: fallbackTakeaway,
    );
  }
}
