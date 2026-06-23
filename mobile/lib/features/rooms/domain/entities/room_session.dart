import 'package:equatable/equatable.dart';

class RoomSession extends Equatable {
  const RoomSession({
    required this.roomId,
    required this.roomName,
    required this.isOwner,
    this.ownerId,
    this.liveKitUrl,
    this.liveKitToken,
  });

  final String roomId;
  final String roomName;
  final bool isOwner;
  final String? ownerId;
  final String? liveKitUrl;
  final String? liveKitToken;

  @override
  List<Object?> get props => [
    roomId,
    roomName,
    isOwner,
    ownerId,
    liveKitUrl,
    liveKitToken,
  ];
}
