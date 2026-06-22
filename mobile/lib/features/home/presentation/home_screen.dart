import 'package:alem_live_mobile/app/theme.dart';
import 'package:alem_live_mobile/core/widgets/grid_background.dart';
import 'package:alem_live_mobile/core/widgets/primary_button.dart';
import 'package:alem_live_mobile/core/widgets/secondary_button.dart';
import 'package:alem_live_mobile/features/auth/presentation/auth_controller.dart';
import 'package:alem_live_mobile/features/home/presentation/widgets/create_room_sheet.dart';
import 'package:alem_live_mobile/features/home/presentation/widgets/join_room_sheet.dart';
import 'package:alem_live_mobile/features/reports/presentation/screens/reports_screen.dart';
import 'package:alem_live_mobile/features/rooms/domain/usecases/rooms_usecases.dart';
import 'package:alem_live_mobile/features/rooms/presentation/room_navigation_args.dart';
import 'package:alem_live_mobile/features/rooms/presentation/screens/room_screen.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

class HomeScreen extends ConsumerWidget {
  const HomeScreen({super.key});

  static const routeName = 'home';
  static const routePath = '/home';

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final user = ref.watch(authControllerProvider).user;

    return Scaffold(
      floatingActionButton: FloatingActionButton(
        onPressed: () => context.go(ReportsScreen.routePath),
        backgroundColor: AppTheme.blue,
        foregroundColor: Colors.white,
        child: const Icon(Icons.description_outlined),
      ),
      body: SafeArea(
        child: GridBackground(
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Row(
                  children: [
                    const _SmallLogo(),
                    const SizedBox(width: 12),
                    Expanded(
                      child: Text(
                        'AlemLive',
                        style: Theme.of(context).textTheme.titleLarge,
                      ),
                    ),
                    IconButton(
                      tooltip: 'Выйти',
                      onPressed: () async {
                        await ref
                            .read(authControllerProvider.notifier)
                            .logout();
                        if (context.mounted) {
                          context.go('/');
                        }
                      },
                      icon: const Icon(Icons.logout_rounded),
                    ),
                  ],
                ),
                const Spacer(),
                Text(
                  'AlemLive',
                  textAlign: TextAlign.center,
                  style: Theme.of(context).textTheme.headlineLarge,
                ),
                const SizedBox(height: 10),
                Text(
                  user == null
                      ? 'Создайте комнату или присоединитесь к встрече.'
                      : '${user.displayName}, создайте комнату или присоединитесь к встрече.',
                  textAlign: TextAlign.center,
                  style: Theme.of(context).textTheme.bodyMedium,
                ),
                const SizedBox(height: 34),
                PrimaryButton(
                  label: 'Создать комнату',
                  icon: Icons.add_rounded,
                  onPressed: () => _openCreateRoomSheet(context, ref),
                ),
                const SizedBox(height: 14),
                SecondaryButton(
                  label: 'Присоединиться',
                  icon: Icons.login_rounded,
                  onPressed: () => _openJoinRoomSheet(context, ref),
                ),
                const Spacer(flex: 2),
              ],
            ),
          ),
        ),
      ),
    );
  }

  void _openCreateRoomSheet(BuildContext context, WidgetRef ref) {
    showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      backgroundColor: Colors.white,
      showDragHandle: true,
      builder: (sheetContext) => CreateRoomSheet(
        onCreate: (roomName, initialMicEnabled, initialCameraEnabled) async {
          try {
            final session = await ref
                .read(roomsUseCasesProvider)
                .createRoom(
                  roomName: roomName,
                  initialMicEnabled: initialMicEnabled,
                  initialCameraEnabled: initialCameraEnabled,
                );
            if (!context.mounted) {
              return;
            }
            Navigator.of(sheetContext).pop();
            context.goNamed(
              RoomScreen.routeName,
              extra: RoomNavigationArgs(
                roomId: session.roomId,
                roomName: session.roomName,
                isOwner: session.isOwner,
                initialMicEnabled: initialMicEnabled,
                initialCameraEnabled: initialCameraEnabled,
                ownerId: session.ownerId,
                liveKitUrl: session.liveKitUrl,
                liveKitToken: session.liveKitToken,
              ),
            );
          } catch (error) {
            if (context.mounted) {
              ScaffoldMessenger.of(
                context,
              ).showSnackBar(SnackBar(content: Text(error.toString())));
            }
          }
        },
      ),
    );
  }

  void _openJoinRoomSheet(BuildContext context, WidgetRef ref) {
    showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      backgroundColor: Colors.white,
      showDragHandle: true,
      builder: (sheetContext) => JoinRoomSheet(
        onJoin: (roomName) async {
          try {
            final session = await ref
                .read(roomsUseCasesProvider)
                .joinRoom(roomName: roomName);
            if (!context.mounted) {
              return;
            }
            Navigator.of(sheetContext).pop();
            context.goNamed(
              RoomScreen.routeName,
              extra: RoomNavigationArgs(
                roomId: session.roomId,
                roomName: session.roomName,
                isOwner: session.isOwner,
                initialMicEnabled: true,
                initialCameraEnabled: true,
                ownerId: session.ownerId,
                liveKitUrl: session.liveKitUrl,
                liveKitToken: session.liveKitToken,
              ),
            );
          } catch (error) {
            if (context.mounted) {
              ScaffoldMessenger.of(
                context,
              ).showSnackBar(SnackBar(content: Text(error.toString())));
            }
          }
        },
      ),
    );
  }
}

class _SmallLogo extends StatelessWidget {
  const _SmallLogo();

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 42,
      height: 42,
      decoration: BoxDecoration(
        color: AppTheme.blue,
        borderRadius: BorderRadius.circular(14),
      ),
      child: const Icon(Icons.video_call_rounded, color: Colors.white),
    );
  }
}
