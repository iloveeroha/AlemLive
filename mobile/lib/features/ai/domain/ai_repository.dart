import 'package:alem_live_mobile/features/ai/domain/entities/ai_answer.dart';
import 'package:alem_live_mobile/features/ai/domain/entities/ai_question.dart';

abstract interface class AiRepository {
  Future<AiAnswer> askForAnswer({
    required AiQuestion question,
    required String fallbackTakeaway,
  });

  Future<String> askQuestion({
    required String reportId,
    required String roomName,
    required String question,
    required String fallbackTakeaway,
  });
}
