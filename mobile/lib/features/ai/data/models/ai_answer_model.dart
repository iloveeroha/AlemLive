import 'package:alem_live_mobile/features/ai/domain/entities/ai_answer.dart';

class AiAnswerModel extends AiAnswer {
  const AiAnswerModel({
    required super.reportId,
    required super.question,
    required super.answer,
    super.language,
  });

  factory AiAnswerModel.fromJson({
    required String reportId,
    required String question,
    required Map<String, dynamic> json,
  }) {
    return AiAnswerModel(
      reportId: reportId,
      question: question,
      answer:
          json['answer']?.toString() ??
          json['text']?.toString() ??
          json['message']?.toString() ??
          '',
      language: json['language']?.toString(),
    );
  }
}
