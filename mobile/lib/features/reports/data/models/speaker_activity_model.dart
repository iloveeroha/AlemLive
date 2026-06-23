import 'package:alem_live_mobile/features/reports/domain/entities/speaker_activity.dart';

class SpeakerActivityModel extends SpeakerActivity {
  const SpeakerActivityModel({
    required super.speakerName,
    required super.talkTime,
    required super.participationPercent,
    required super.isMostActive,
  });

  factory SpeakerActivityModel.fromJson(Map<String, dynamic> json) {
    final percent = _readInt(json['participationPercent'] ?? json['talk']);
    final talkTimeSeconds =
        _readInt(json['talkTimeSeconds']) ?? _readTalkTimeText(json);
    return SpeakerActivityModel(
      speakerName: (json['speakerName'] ?? json['name'] ?? 'Speaker')
          .toString(),
      talkTime: Duration(seconds: talkTimeSeconds ?? (percent ?? 0) * 60),
      participationPercent: percent ?? 0,
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

  static int? _readInt(Object? value) {
    if (value is int) {
      return value;
    }
    if (value is num) {
      return value.round();
    }
    return int.tryParse(value?.toString() ?? '');
  }

  static int? _readTalkTimeText(Map<String, dynamic> json) {
    final value = json['talkTimeText']?.toString();
    if (value == null || value.trim().isEmpty) {
      return null;
    }

    final parts = value.split(':').map(int.tryParse).toList();
    if (parts.length == 2 && parts.every((part) => part != null)) {
      return (parts[0]! * 60) + parts[1]!;
    }
    if (parts.length == 3 && parts.every((part) => part != null)) {
      return (parts[0]! * 3600) + (parts[1]! * 60) + parts[2]!;
    }
    return null;
  }
}
