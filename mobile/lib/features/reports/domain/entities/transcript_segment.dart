import 'package:equatable/equatable.dart';

class TranscriptSegment extends Equatable {
  const TranscriptSegment({
    required this.timecode,
    required this.speakerName,
    required this.text,
  });

  final String timecode;
  final String speakerName;
  final String text;

  @override
  List<Object?> get props => [timecode, speakerName, text];
}
