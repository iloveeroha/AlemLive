import 'package:equatable/equatable.dart';

class SpeakerActivity extends Equatable {
  const SpeakerActivity({
    required this.speakerName,
    required this.talkTime,
    required this.participationPercent,
    required this.isMostActive,
  });

  final String speakerName;
  final Duration talkTime;
  final int participationPercent;
  final bool isMostActive;

  String get talkTimeLabel {
    final minutes = talkTime.inMinutes;
    final seconds = talkTime.inSeconds.remainder(60).toString().padLeft(2, '0');
    return '$minutes:$seconds';
  }

  @override
  List<Object?> get props => [
    speakerName,
    talkTime,
    participationPercent,
    isMostActive,
  ];
}
