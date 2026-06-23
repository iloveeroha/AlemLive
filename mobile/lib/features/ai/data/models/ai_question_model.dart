import 'package:alem_live_mobile/features/ai/domain/entities/ai_question.dart';

class AiQuestionModel extends AiQuestion {
  const AiQuestionModel({
    required super.reportId,
    required super.question,
    super.roomName,
  });

  factory AiQuestionModel.fromEntity(AiQuestion question) {
    return AiQuestionModel(
      reportId: question.reportId,
      question: question.question,
      roomName: question.roomName,
    );
  }

  Map<String, dynamic> toAskAiJson() {
    return {
      'question': question,
      if (roomName != null && roomName!.trim().isNotEmpty) 'roomName': roomName,
    };
  }

  Map<String, dynamic> toChatJson() {
    return {'message': question};
  }
}
