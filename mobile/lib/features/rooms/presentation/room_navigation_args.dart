import 'package:equatable/equatable.dart';

class RoomNavigationArgs extends Equatable {
  const RoomNavigationArgs({
    required this.roomId,
    required this.roomName,
    required this.isOwner,
    required this.initialMicEnabled,
    required this.initialCameraEnabled,
    this.ownerId,
    this.liveKitUrl,
    this.liveKitToken,
  });

  final String roomId;
  final String roomName;
  final bool isOwner;
  final bool initialMicEnabled;
  final bool initialCameraEnabled;
  final String? ownerId;
  final String? liveKitUrl;
  final String? liveKitToken;

  bool get hasLiveKitCredentials {
    return (liveKitUrl?.trim().isNotEmpty ?? false) &&
        (liveKitToken?.trim().isNotEmpty ?? false);
  }

  @override
  List<Object?> get props => [
    roomId,
    roomName,
    isOwner,
    initialMicEnabled,
    initialCameraEnabled,
    ownerId,
    liveKitUrl,
    liveKitToken,
  ];
}
