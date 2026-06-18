import 'package:alem_live_mobile/features/reports/domain/entities/speaker_activity.dart';

class SpeakerActivityModel extends SpeakerActivity {
  const SpeakerActivityModel({
    required super.speakerName,
    required super.talkTime,
    required super.participationPercent,
    required super.isMostActive,
  });

  factory SpeakerActivityModel.fromJson(Map<String, dynamic> json) {
    return SpeakerActivityModel(
      speakerName: json['speakerName'] as String,
      talkTime: Duration(seconds: json['talkTimeSeconds'] as int),
      participationPercent: json['participationPercent'] as int,
      isMostActive: json['isMostActive'] as bool? ?? false,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'speakerName': speakerName,
      'talkTimeSeconds': talkTime.inSeconds,
      'participationPercent': participationPercent,
      'isMostActive': isMostActive,
    };
  }
}
