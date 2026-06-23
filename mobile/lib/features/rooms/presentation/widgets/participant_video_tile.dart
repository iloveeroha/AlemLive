import 'package:alem_live_mobile/app/theme.dart';
import 'package:alem_live_mobile/features/rooms/domain/entities/room_participant.dart';
import 'package:alem_live_mobile/features/rooms/presentation/livekit/livekit_room_state.dart';
import 'package:flutter/material.dart';
import 'package:livekit_client/livekit_client.dart';

class ParticipantVideoTile extends StatelessWidget {
  const ParticipantVideoTile({required this.participantView, super.key});

  final RoomParticipantView participantView;

  RoomParticipant get participant => participantView.participant;

  @override
  Widget build(BuildContext context) {
    return Container(
      clipBehavior: Clip.antiAlias,
      decoration: BoxDecoration(
        color: participant.isCameraEnabled
            ? const Color(0xFFEFF6FF)
            : const Color(0xFF111827),
        borderRadius: BorderRadius.circular(20),
        border: Border.all(
          color: participant.isCurrentUser ? AppTheme.blue : AppTheme.border,
          width: participant.isCurrentUser ? 2 : 1,
        ),
        boxShadow: [
          BoxShadow(
            color: Colors.black.withValues(alpha: 0.07),
            blurRadius: 18,
            offset: const Offset(0, 10),
          ),
        ],
      ),
      child: Stack(
        children: [
          Positioned.fill(
            child: _VideoContent(participantView: participantView),
          ),
          if (participant.isScreenSharing)
            Positioned(
              top: 10,
              left: participant.isOwner ? 112 : 10,
              child: const _Badge(
                icon: Icons.screen_share_rounded,
                label: 'Экран',
                backgroundColor: Colors.black,
              ),
            ),
          Positioned(
            left: 10,
            right: 10,
            bottom: 10,
            child: _ParticipantMeta(participant: participant),
          ),
          if (participant.isOwner)
            Positioned(
              top: 10,
              left: 10,
              child: _Badge(
                icon: Icons.star_rounded,
                label: 'Создатель',
                backgroundColor: AppTheme.blue,
              ),
            ),
          Positioned(
            top: 10,
            right: 10,
            child: Row(
              children: [
                _RoundStatusIcon(
                  icon: participant.isMicEnabled
                      ? Icons.mic_rounded
                      : Icons.mic_off_rounded,
                  active: participant.isMicEnabled,
                ),
                const SizedBox(width: 6),
                _RoundStatusIcon(
                  icon: participant.isCameraEnabled
                      ? Icons.videocam_rounded
                      : Icons.videocam_off_rounded,
                  active: participant.isCameraEnabled,
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _VideoContent extends StatelessWidget {
  const _VideoContent({required this.participantView});

  final RoomParticipantView participantView;

  RoomParticipant get participant => participantView.participant;

  @override
  Widget build(BuildContext context) {
    if (participantView.canRenderVideo) {
      return ColoredBox(
        color: Colors.black,
        child: VideoTrackRenderer(
          participantView.videoTrack!,
          renderMode: VideoRenderMode.auto,
        ),
      );
    }

    if (!participant.isCameraEnabled && !participant.isScreenSharing) {
      return Center(
        child: CircleAvatar(
          radius: 34,
          backgroundColor: AppTheme.blue,
          child: Text(
            participant.initials,
            style: const TextStyle(
              color: Colors.white,
              fontWeight: FontWeight.w800,
              fontSize: 24,
            ),
          ),
        ),
      );
    }

    return DecoratedBox(
      decoration: const BoxDecoration(
        gradient: LinearGradient(
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
          colors: [Color(0xFFEFF6FF), Color(0xFFFFFFFF)],
        ),
      ),
      child: Center(
        child: Icon(
          Icons.person_rounded,
          color: AppTheme.blue.withValues(alpha: 0.34),
          size: 72,
        ),
      ),
    );
  }
}

class _ParticipantMeta extends StatelessWidget {
  const _ParticipantMeta({required this.participant});

  final RoomParticipant participant;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
      decoration: BoxDecoration(
        color: Colors.black.withValues(alpha: 0.58),
        borderRadius: BorderRadius.circular(12),
      ),
      child: Text(
        participant.isCurrentUser
            ? '${participant.name} (вы)'
            : participant.name,
        maxLines: 1,
        overflow: TextOverflow.ellipsis,
        style: const TextStyle(
          color: Colors.white,
          fontWeight: FontWeight.w700,
        ),
      ),
    );
  }
}

class _Badge extends StatelessWidget {
  const _Badge({
    required this.icon,
    required this.label,
    required this.backgroundColor,
  });

  final IconData icon;
  final String label;
  final Color backgroundColor;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 9, vertical: 6),
      decoration: BoxDecoration(
        color: backgroundColor,
        borderRadius: BorderRadius.circular(999),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(icon, color: Colors.white, size: 14),
          const SizedBox(width: 4),
          Text(
            label,
            style: const TextStyle(
              color: Colors.white,
              fontSize: 11,
              fontWeight: FontWeight.w800,
            ),
          ),
        ],
      ),
    );
  }
}

class _RoundStatusIcon extends StatelessWidget {
  const _RoundStatusIcon({required this.icon, required this.active});

  final IconData icon;
  final bool active;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 30,
      height: 30,
      decoration: BoxDecoration(
        color: active ? Colors.black.withValues(alpha: 0.72) : AppTheme.blue,
        shape: BoxShape.circle,
      ),
      child: Icon(icon, color: Colors.white, size: 16),
    );
  }
}
