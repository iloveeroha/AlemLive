import 'package:alem_live_mobile/app/theme.dart';
import 'package:alem_live_mobile/features/rooms/domain/entities/room_participant.dart';
import 'package:alem_live_mobile/features/rooms/presentation/livekit/livekit_room_state.dart';
import 'package:flutter/material.dart';

class ParticipantsBottomSheet extends StatelessWidget {
  const ParticipantsBottomSheet({
    required this.participants,
    required this.isCurrentUserOwner,
    required this.currentUserId,
    required this.isMicControlLoading,
    required this.isCameraControlLoading,
    required this.onToggleMic,
    required this.onToggleCamera,
    super.key,
  });

  final List<RoomParticipantView> participants;
  final bool isCurrentUserOwner;
  final String currentUserId;
  final bool Function(String participantId) isMicControlLoading;
  final bool Function(String participantId) isCameraControlLoading;
  final ValueChanged<String> onToggleMic;
  final ValueChanged<String> onToggleCamera;

  @override
  Widget build(BuildContext context) {
    return SafeArea(
      child: Padding(
        padding: const EdgeInsets.fromLTRB(20, 0, 20, 24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('Участники', style: Theme.of(context).textTheme.titleLarge),
            const SizedBox(height: 12),
            Flexible(
              child: ListView.separated(
                shrinkWrap: true,
                itemCount: participants.length,
                separatorBuilder: (_, _) => const Divider(height: 1),
                itemBuilder: (context, index) {
                  final participant = participants[index].participant;
                  return _ParticipantRow(
                    participant: participant,
                    canManage:
                        isCurrentUserOwner && participant.id != currentUserId,
                    isMicLoading: isMicControlLoading(participant.id),
                    isCameraLoading: isCameraControlLoading(participant.id),
                    onToggleMic: () => onToggleMic(participant.id),
                    onToggleCamera: () => onToggleCamera(participant.id),
                  );
                },
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _ParticipantRow extends StatelessWidget {
  const _ParticipantRow({
    required this.participant,
    required this.canManage,
    required this.isMicLoading,
    required this.isCameraLoading,
    required this.onToggleMic,
    required this.onToggleCamera,
  });

  final RoomParticipant participant;
  final bool canManage;
  final bool isMicLoading;
  final bool isCameraLoading;
  final VoidCallback onToggleMic;
  final VoidCallback onToggleCamera;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 12),
      child: Row(
        children: [
          CircleAvatar(
            backgroundColor: AppTheme.blue,
            child: Text(
              participant.initials,
              style: const TextStyle(
                color: Colors.white,
                fontWeight: FontWeight.w800,
              ),
            ),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  participant.isCurrentUser
                      ? '${participant.name} (вы)'
                      : participant.name,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: Theme.of(context).textTheme.titleMedium,
                ),
                const SizedBox(height: 4),
                Wrap(
                  spacing: 8,
                  runSpacing: 6,
                  children: [
                    _StateLabel(
                      icon: participant.isMicEnabled
                          ? Icons.mic_rounded
                          : Icons.mic_off_rounded,
                      label: participant.isMicEnabled
                          ? 'Микрофон'
                          : 'Без звука',
                    ),
                    _StateLabel(
                      icon: participant.isCameraEnabled
                          ? Icons.videocam_rounded
                          : Icons.videocam_off_rounded,
                      label: participant.isCameraEnabled
                          ? 'Камера'
                          : 'Без видео',
                    ),
                    if (participant.isOwner)
                      const _StateLabel(
                        icon: Icons.star_rounded,
                        label: 'Создатель',
                      ),
                  ],
                ),
              ],
            ),
          ),
          if (canManage) ...[
            IconButton.filled(
              tooltip: participant.isMicEnabled
                  ? 'Выключить микрофон'
                  : 'Запросить микрофон',
              onPressed: isMicLoading ? null : onToggleMic,
              style: IconButton.styleFrom(backgroundColor: AppTheme.blue),
              icon: isMicLoading
                  ? const _MiniLoader()
                  : Icon(
                      participant.isMicEnabled
                          ? Icons.mic_off_rounded
                          : Icons.mic_rounded,
                    ),
            ),
            const SizedBox(width: 6),
            IconButton.filled(
              tooltip: participant.isCameraEnabled
                  ? 'Выключить камеру'
                  : 'Запросить камеру',
              onPressed: isCameraLoading ? null : onToggleCamera,
              style: IconButton.styleFrom(backgroundColor: AppTheme.blue),
              icon: isCameraLoading
                  ? const _MiniLoader()
                  : Icon(
                      participant.isCameraEnabled
                          ? Icons.videocam_off_rounded
                          : Icons.videocam_rounded,
                    ),
            ),
          ],
        ],
      ),
    );
  }
}

class _MiniLoader extends StatelessWidget {
  const _MiniLoader();

  @override
  Widget build(BuildContext context) {
    return const SizedBox(
      width: 18,
      height: 18,
      child: CircularProgressIndicator(color: Colors.white, strokeWidth: 2),
    );
  }
}

class _StateLabel extends StatelessWidget {
  const _StateLabel({required this.icon, required this.label});

  final IconData icon;
  final String label;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 5),
      decoration: BoxDecoration(
        color: const Color(0xFFF2F4F7),
        borderRadius: BorderRadius.circular(999),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(icon, size: 13, color: AppTheme.muted),
          const SizedBox(width: 4),
          Text(
            label,
            style: const TextStyle(
              color: AppTheme.muted,
              fontSize: 12,
              fontWeight: FontWeight.w700,
            ),
          ),
        ],
      ),
    );
  }
}
