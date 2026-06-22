import 'package:alem_live_mobile/features/reports/domain/entities/transcript_segment.dart';

class TranscriptSegmentModel extends TranscriptSegment {
  const TranscriptSegmentModel({
    required super.timecode,
    required super.speakerName,
    required super.text,
  });

  factory TranscriptSegmentModel.fromJson(Map<String, dynamic> json) {
    final rawTimecode = json['timecode'] ?? json['time'] ?? json['start'] ?? '';
    return TranscriptSegmentModel(
      timecode: _formatTimecode(rawTimecode),
      speakerName: (json['speakerName'] ?? json['speaker'] ?? 'Speaker')
          .toString(),
      text: (json['text'] ?? '').toString(),
    );
  }

  Map<String, dynamic> toJson() {
    return {'timecode': timecode, 'speakerName': speakerName, 'text': text};
  }

  static String _formatTimecode(Object? value) {
    if (value is num) {
      final totalSeconds = value.round();
      final hours = totalSeconds ~/ 3600;
      final minutes = (totalSeconds % 3600) ~/ 60;
      final seconds = totalSeconds % 60;
      if (hours > 0) {
        return '${hours.toString().padLeft(2, '0')}:'
            '${minutes.toString().padLeft(2, '0')}:'
            '${seconds.toString().padLeft(2, '0')}';
      }
      return '${minutes.toString().padLeft(2, '0')}:'
          '${seconds.toString().padLeft(2, '0')}';
    }

    return value?.toString() ?? '';
  }
}
