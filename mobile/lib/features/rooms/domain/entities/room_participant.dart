import 'package:equatable/equatable.dart';

class RoomParticipant extends Equatable {
  const RoomParticipant({
    required this.id,
    required this.name,
    required this.isCurrentUser,
    required this.isOwner,
    required this.isMicEnabled,
    required this.isCameraEnabled,
    this.isScreenSharing = false,
  });

  final String id;
  final String name;
  final bool isCurrentUser;
  final bool isOwner;
  final bool isMicEnabled;
  final bool isCameraEnabled;
  final bool isScreenSharing;

  String get initials {
    final parts = name.trim().split(RegExp(r'\s+'));
    if (parts.isEmpty || parts.first.isEmpty) {
      return '?';
    }
    return parts.take(2).map((part) => part[0].toUpperCase()).join();
  }

  RoomParticipant copyWith({
    bool? isMicEnabled,
    bool? isCameraEnabled,
    bool? isOwner,
    bool? isScreenSharing,
  }) {
    return RoomParticipant(
      id: id,
      name: name,
      isCurrentUser: isCurrentUser,
      isOwner: isOwner ?? this.isOwner,
      isMicEnabled: isMicEnabled ?? this.isMicEnabled,
      isCameraEnabled: isCameraEnabled ?? this.isCameraEnabled,
      isScreenSharing: isScreenSharing ?? this.isScreenSharing,
    );
  }

  @override
  List<Object?> get props => [
    id,
    name,
    isCurrentUser,
    isOwner,
    isMicEnabled,
    isCameraEnabled,
    isScreenSharing,
  ];
}
