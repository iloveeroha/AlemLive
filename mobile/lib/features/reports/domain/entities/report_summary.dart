import 'package:equatable/equatable.dart';

class ReportSummary extends Equatable {
  const ReportSummary({
    required this.text,
    required this.topics,
    required this.takeaways,
  });

  final String text;
  final List<String> topics;
  final List<String> takeaways;

  bool get isEmpty {
    return text.trim().isEmpty && topics.isEmpty && takeaways.isEmpty;
  }

  @override
  List<Object?> get props => [text, topics, takeaways];
}
