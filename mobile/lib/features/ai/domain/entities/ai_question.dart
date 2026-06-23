import 'package:equatable/equatable.dart';

class AiQuestion extends Equatable {
  const AiQuestion({
    required this.reportId,
    required this.question,
    this.roomName,
  });

  final String reportId;
  final String question;
  final String? roomName;

  @override
  List<Object?> get props => [reportId, question, roomName];
}
