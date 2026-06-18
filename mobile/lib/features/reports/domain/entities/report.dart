import 'package:alem_live_mobile/features/reports/domain/entities/action_item.dart';
import 'package:alem_live_mobile/features/reports/domain/entities/speaker_activity.dart';
import 'package:alem_live_mobile/features/reports/domain/entities/transcript_segment.dart';
import 'package:equatable/equatable.dart';

enum ReportProcessingStatus { processing, ready, error }

class Report extends Equatable {
  const Report({
    required this.id,
    required this.roomName,
    required this.startedAt,
    required this.duration,
    required this.status,
    required this.summary,
    required this.topics,
    required this.takeaways,
    required this.actionItems,
    required this.speakerActivity,
    required this.transcript,
  });

  final String id;
  final String roomName;
  final DateTime startedAt;
  final Duration duration;
  final ReportProcessingStatus status;
  final String summary;
  final List<String> topics;
  final List<String> takeaways;
  final List<ActionItem> actionItems;
  final List<SpeakerActivity> speakerActivity;
  final List<TranscriptSegment> transcript;

  String get statusLabel {
    return switch (status) {
      ReportProcessingStatus.processing => 'Обрабатывается',
      ReportProcessingStatus.ready => 'Готово',
      ReportProcessingStatus.error => 'Ошибка',
    };
  }

  String get durationLabel {
    final hours = duration.inHours;
    final minutes = duration.inMinutes.remainder(60);
    if (hours > 0) {
      return '$hours ч ${minutes.toString().padLeft(2, '0')} мин';
    }
    return '$minutes мин';
  }

  String get startedAtLabel {
    final day = startedAt.day.toString().padLeft(2, '0');
    final month = startedAt.month.toString().padLeft(2, '0');
    final year = startedAt.year;
    final hour = startedAt.hour.toString().padLeft(2, '0');
    final minute = startedAt.minute.toString().padLeft(2, '0');
    return '$day.$month.$year, $hour:$minute';
  }

  @override
  List<Object?> get props => [
    id,
    roomName,
    startedAt,
    duration,
    status,
    summary,
    topics,
    takeaways,
    actionItems,
    speakerActivity,
    transcript,
  ];
}
