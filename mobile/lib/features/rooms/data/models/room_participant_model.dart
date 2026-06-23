import 'package:alem_live_mobile/features/rooms/domain/entities/room_participant.dart';

class RoomParticipantModel extends RoomParticipant {
  const RoomParticipantModel({
    required super.id,
    required super.name,
    required super.isCurrentUser,
    required super.isOwner,
    required super.isMicEnabled,
    required super.isCameraEnabled,
  });

  factory RoomParticipantModel.fromJson(Map<String, dynamic> json) {
    return RoomParticipantModel(
      id: json['id'] as String,
      name: json['name'] as String,
      isCurrentUser: json['isCurrentUser'] as bool? ?? false,
      isOwner: json['isOwner'] as bool? ?? false,
      isMicEnabled: json['isMicEnabled'] as bool? ?? false,
      isCameraEnabled: json['isCameraEnabled'] as bool? ?? false,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'name': name,
      'isCurrentUser': isCurrentUser,
      'isOwner': isOwner,
      'isMicEnabled': isMicEnabled,
      'isCameraEnabled': isCameraEnabled,
    };
  }
}
