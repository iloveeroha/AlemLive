import 'package:alem_live_mobile/features/reports/domain/entities/transcript_segment.dart';

class TranscriptSegmentModel extends TranscriptSegment {
  const TranscriptSegmentModel({
    required super.timecode,
    required super.speakerName,
    required super.text,
  });

  factory TranscriptSegmentModel.fromJson(Map<String, dynamic> json) {
    return TranscriptSegmentModel(
      timecode: json['timecode'] as String,
      speakerName: json['speakerName'] as String,
      text: json['text'] as String,
    );
  }

  Map<String, dynamic> toJson() {
    return {'timecode': timecode, 'speakerName': speakerName, 'text': text};
  }
}
