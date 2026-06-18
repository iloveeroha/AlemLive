import 'package:alem_live_mobile/app/theme.dart';
import 'package:alem_live_mobile/features/home/presentation/home_screen.dart';
import 'package:alem_live_mobile/features/rooms/presentation/livekit/livekit_room_controller.dart';
import 'package:alem_live_mobile/features/rooms/presentation/livekit/livekit_room_state.dart';
import 'package:alem_live_mobile/features/rooms/presentation/room_navigation_args.dart';
import 'package:alem_live_mobile/features/rooms/presentation/widgets/chat_overlay.dart';
import 'package:alem_live_mobile/features/rooms/presentation/widgets/leave_room_dialog.dart';
import 'package:alem_live_mobile/features/rooms/presentation/widgets/participants_bottom_sheet.dart';
import 'package:alem_live_mobile/features/rooms/presentation/widgets/room_control_bar.dart';
import 'package:alem_live_mobile/features/rooms/presentation/widgets/room_video_grid.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

class RoomScreen extends ConsumerStatefulWidget {
  const RoomScreen({required this.args, super.key});

  static const routeName = 'room';
  static const routePath = '/room';

  final RoomNavigationArgs args;

  @override
  ConsumerState<RoomScreen> createState() => _RoomScreenState();
}

class _RoomScreenState extends ConsumerState<RoomScreen> {
  bool _chatOpen = false;
  String? _lastNotice;
  String? _lastControlError;

  @override
  Widget build(BuildContext context) {
    final roomController = ref.watch(
      liveKitRoomControllerProvider(widget.args),
    );

    return ListenableBuilder(
      listenable: roomController,
      builder: (context, _) {
        final roomState = roomController.state;
        _showStateMessages(roomState);

        return Scaffold(
          backgroundColor: const Color(0xFFF8FAFF),
          body: SafeArea(
            child: Column(
              children: [
                _RoomHeader(
                  roomName: widget.args.roomName,
                  isOwner: roomState.isCurrentUserOwner,
                  isRecording: roomState.isRecording,
                  isScreenSharing: roomState.screenSharing,
                  status: roomState.status,
                  onLeave: _confirmLeave,
                ),
                Expanded(
                  child: Padding(
                    padding: const EdgeInsets.fromLTRB(12, 0, 12, 12),
                    child: Stack(
                      children: [
                        RoomVideoGrid(participants: roomState.participants),
                        if (roomState.isLoading)
                          const Positioned.fill(
                            child: _ConnectionOverlay(
                              message: 'Подключаемся к LiveKit...',
                              showProgress: true,
                            ),
                          ),
                        if (roomState.hasError)
                          Positioned.fill(
                            child: _ConnectionOverlay(
                              message:
                                  roomState.errorMessage ??
                                  'Не удалось подключиться к LiveKit.',
                              actionLabel: 'Повторить',
                              onAction: roomController.connect,
                            ),
                          ),
                        if (roomState.status == LiveKitRoomStatus.reconnecting)
                          const Positioned(
                            left: 12,
                            right: 12,
                            top: 12,
                            child: _RoomStatusBanner(
                              text: 'Восстанавливаем подключение...',
                            ),
                          ),
                        if (roomState.screenSharing)
                          const Positioned(
                            left: 12,
                            right: 12,
                            bottom: 12,
                            child: _RoomStatusBanner(
                              text: 'Вы показываете экран',
                            ),
                          ),
                        if (_chatOpen)
                          Positioned.fill(
                            child: ChatOverlay(
                              messages: roomState.messages,
                              onSend: roomController.sendChatMessage,
                              onClose: _toggleChat,
                            ),
                          ),
                      ],
                    ),
                  ),
                ),
                RoomControlBar(
                  micEnabled: roomState.micEnabled,
                  cameraEnabled: roomState.cameraEnabled,
                  chatOpen: _chatOpen,
                  screenSharing: roomState.screenSharing,
                  recording: roomState.isRecording,
                  onToggleMic: roomController.toggleMicrophone,
                  onToggleCamera: roomController.toggleCamera,
                  onToggleChat: _toggleChat,
                  onShowParticipants: _showParticipants,
                  onToggleScreenShare: roomController.toggleScreenShare,
                  onToggleRecording: roomController.toggleRecording,
                  onLeave: _confirmLeave,
                ),
              ],
            ),
          ),
        );
      },
    );
  }

  void _toggleChat() {
    setState(() => _chatOpen = !_chatOpen);
  }

  void _showParticipants() {
    final controller = ref.read(liveKitRoomControllerProvider(widget.args));
    final roomState = controller.state;
    var currentUserId = '';
    for (final view in roomState.participants) {
      if (view.participant.isCurrentUser) {
        currentUserId = view.participant.id;
        break;
      }
    }

    showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      backgroundColor: Colors.white,
      showDragHandle: true,
      builder: (context) => ListenableBuilder(
        listenable: controller,
        builder: (context, _) {
          final latestState = controller.state;
          return ParticipantsBottomSheet(
            participants: latestState.participants,
            isCurrentUserOwner: latestState.isCurrentUserOwner,
            currentUserId: currentUserId,
            isMicControlLoading: (participantId) =>
                latestState.isControlLoading(
                  participantId,
                  ParticipantControlType.microphone,
                ),
            isCameraControlLoading: (participantId) => latestState
                .isControlLoading(participantId, ParticipantControlType.camera),
            onToggleMic: controller.toggleParticipantMicrophone,
            onToggleCamera: controller.toggleParticipantCamera,
          );
        },
      ),
    );
  }

  void _showStateMessages(LiveKitRoomState roomState) {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) {
        return;
      }
      final notice = roomState.roomNotice;
      if (notice != null && notice != _lastNotice) {
        _lastNotice = notice;
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text(notice)));
      }

      final controlError = roomState.controlErrorMessage;
      if (controlError != null && controlError != _lastControlError) {
        _lastControlError = controlError;
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text(controlError)));
      }
    });
  }

  Future<void> _confirmLeave() async {
    final shouldLeave = await showDialog<bool>(
      context: context,
      builder: (context) => const LeaveRoomDialog(),
    );
    if (shouldLeave == true && mounted) {
      await ref.read(liveKitRoomControllerProvider(widget.args)).leaveRoom();
      if (!mounted) {
        return;
      }
      context.go(HomeScreen.routePath);
    }
  }
}

class _RoomHeader extends StatelessWidget {
  const _RoomHeader({
    required this.roomName,
    required this.isOwner,
    required this.isRecording,
    required this.isScreenSharing,
    required this.status,
    required this.onLeave,
  });

  final String roomName;
  final bool isOwner;
  final bool isRecording;
  final bool isScreenSharing;
  final LiveKitRoomStatus status;
  final VoidCallback onLeave;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 14, 16, 12),
      child: Row(
        children: [
          Container(
            width: 42,
            height: 42,
            decoration: BoxDecoration(
              color: AppTheme.blue,
              borderRadius: BorderRadius.circular(14),
            ),
            child: const Icon(Icons.video_call_rounded, color: Colors.white),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  roomName,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: Theme.of(context).textTheme.titleLarge,
                ),
                Wrap(
                  spacing: 8,
                  runSpacing: 4,
                  children: [
                    if (isOwner) const _StatusChip(label: 'Вы создатель'),
                    if (status == LiveKitRoomStatus.mock)
                      const _StatusChip(label: 'Mock'),
                    if (status == LiveKitRoomStatus.connected)
                      const _StatusChip(label: 'LiveKit'),
                    if (isScreenSharing)
                      const _StatusChip(label: 'Демонстрация экрана'),
                  ],
                ),
              ],
            ),
          ),
          if (isRecording)
            const _StatusChip(label: 'REC', color: Color(0xFFE11D48)),
          const SizedBox(width: 8),
          IconButton.filled(
            tooltip: 'Выйти из комнаты',
            onPressed: onLeave,
            style: IconButton.styleFrom(
              backgroundColor: const Color(0xFFE11D48),
              foregroundColor: Colors.white,
            ),
            icon: const Icon(Icons.call_end_rounded),
          ),
        ],
      ),
    );
  }
}

class _StatusChip extends StatelessWidget {
  const _StatusChip({required this.label, this.color = AppTheme.blue});

  final String label;
  final Color color;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.12),
        borderRadius: BorderRadius.circular(999),
      ),
      child: Text(
        label,
        style: TextStyle(
          color: color,
          fontSize: 12,
          fontWeight: FontWeight.w800,
        ),
      ),
    );
  }
}

class _ConnectionOverlay extends StatelessWidget {
  const _ConnectionOverlay({
    required this.message,
    this.showProgress = false,
    this.actionLabel,
    this.onAction,
  });

  final String message;
  final bool showProgress;
  final String? actionLabel;
  final VoidCallback? onAction;

  @override
  Widget build(BuildContext context) {
    return DecoratedBox(
      decoration: BoxDecoration(color: Colors.white.withValues(alpha: 0.86)),
      child: Center(
        child: Container(
          constraints: const BoxConstraints(maxWidth: 320),
          padding: const EdgeInsets.all(18),
          decoration: BoxDecoration(
            color: Colors.white,
            border: Border.all(color: AppTheme.border),
            borderRadius: BorderRadius.circular(18),
            boxShadow: [
              BoxShadow(
                color: Colors.black.withValues(alpha: 0.08),
                blurRadius: 20,
                offset: const Offset(0, 12),
              ),
            ],
          ),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              if (showProgress) ...[
                const CircularProgressIndicator(color: AppTheme.blue),
                const SizedBox(height: 14),
              ],
              Text(
                message,
                textAlign: TextAlign.center,
                style: Theme.of(context).textTheme.titleMedium,
              ),
              if (actionLabel != null && onAction != null) ...[
                const SizedBox(height: 14),
                FilledButton.icon(
                  onPressed: onAction,
                  icon: const Icon(Icons.refresh_rounded),
                  label: Text(actionLabel!),
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }
}

class _RoomStatusBanner extends StatelessWidget {
  const _RoomStatusBanner({required this.text});

  final String text;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
      decoration: BoxDecoration(
        color: AppTheme.blue,
        borderRadius: BorderRadius.circular(14),
        boxShadow: [
          BoxShadow(
            color: AppTheme.blue.withValues(alpha: 0.24),
            blurRadius: 16,
            offset: const Offset(0, 8),
          ),
        ],
      ),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          const SizedBox(
            width: 16,
            height: 16,
            child: CircularProgressIndicator(
              color: Colors.white,
              strokeWidth: 2,
            ),
          ),
          const SizedBox(width: 8),
          Flexible(
            child: Text(
              text,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: const TextStyle(
                color: Colors.white,
                fontWeight: FontWeight.w800,
              ),
            ),
          ),
        ],
      ),
    );
  }
}
