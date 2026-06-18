import 'package:alem_live_mobile/app/theme.dart';
import 'package:alem_live_mobile/features/rooms/presentation/widgets/room_control_button.dart';
import 'package:flutter/material.dart';

class RoomControlBar extends StatelessWidget {
  const RoomControlBar({
    required this.micEnabled,
    required this.cameraEnabled,
    required this.chatOpen,
    required this.screenSharing,
    required this.recording,
    required this.onToggleMic,
    required this.onToggleCamera,
    required this.onToggleChat,
    required this.onShowParticipants,
    required this.onToggleScreenShare,
    required this.onToggleRecording,
    required this.onLeave,
    super.key,
  });

  final bool micEnabled;
  final bool cameraEnabled;
  final bool chatOpen;
  final bool screenSharing;
  final bool recording;
  final VoidCallback onToggleMic;
  final VoidCallback onToggleCamera;
  final VoidCallback onToggleChat;
  final VoidCallback onShowParticipants;
  final VoidCallback onToggleScreenShare;
  final VoidCallback onToggleRecording;
  final VoidCallback onLeave;

  @override
  Widget build(BuildContext context) {
    return SafeArea(
      top: false,
      child: Container(
        padding: const EdgeInsets.fromLTRB(6, 8, 6, 10),
        decoration: const BoxDecoration(
          color: Colors.white,
          border: Border(top: BorderSide(color: AppTheme.border)),
        ),
        child: Row(
          children: [
            Expanded(
              child: RoomControlButton(
                icon: micEnabled ? Icons.mic_rounded : Icons.mic_off_rounded,
                label: 'Мик',
                backgroundColor: micEnabled ? Colors.black : AppTheme.blue,
                onPressed: onToggleMic,
              ),
            ),
            Expanded(
              child: RoomControlButton(
                icon: cameraEnabled
                    ? Icons.videocam_rounded
                    : Icons.videocam_off_rounded,
                label: 'Видео',
                backgroundColor: cameraEnabled ? Colors.black : AppTheme.blue,
                onPressed: onToggleCamera,
              ),
            ),
            Expanded(
              child: RoomControlButton(
                icon: Icons.chat_bubble_outline_rounded,
                label: 'Чат',
                backgroundColor: chatOpen ? AppTheme.blue : Colors.black,
                onPressed: onToggleChat,
              ),
            ),
            Expanded(
              child: RoomControlButton(
                icon: Icons.people_alt_outlined,
                label: 'Люди',
                backgroundColor: Colors.black,
                onPressed: onShowParticipants,
              ),
            ),
            Expanded(
              child: RoomControlButton(
                icon: Icons.screen_share_outlined,
                label: 'Экран',
                backgroundColor: screenSharing ? AppTheme.blue : Colors.black,
                onPressed: onToggleScreenShare,
              ),
            ),
            Expanded(
              child: RoomControlButton(
                icon: recording
                    ? Icons.stop_circle_outlined
                    : Icons.fiber_manual_record_rounded,
                label: recording ? 'REC' : 'Запись',
                backgroundColor: recording
                    ? const Color(0xFFE11D48)
                    : Colors.black,
                onPressed: onToggleRecording,
              ),
            ),
            Expanded(
              child: RoomControlButton(
                icon: Icons.call_end_rounded,
                label: 'Выйти',
                backgroundColor: const Color(0xFFE11D48),
                onPressed: onLeave,
              ),
            ),
          ],
        ),
      ),
    );
  }
}
