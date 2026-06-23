import 'package:equatable/equatable.dart';

class AiAnswer extends Equatable {
  const AiAnswer({
    required this.reportId,
    required this.question,
    required this.answer,
    this.language,
  });

  final String reportId;
  final String question;
  final String answer;
  final String? language;

  @override
  List<Object?> get props => [reportId, question, answer, language];
}
