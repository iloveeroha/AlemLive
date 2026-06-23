import 'package:alem_live_mobile/features/auth/presentation/login_screen.dart';
import 'package:alem_live_mobile/features/auth/presentation/register_screen.dart';
import 'package:alem_live_mobile/features/home/presentation/home_screen.dart';
import 'package:alem_live_mobile/features/reports/presentation/screens/report_detail_screen.dart';
import 'package:alem_live_mobile/features/reports/presentation/screens/reports_screen.dart';
import 'package:alem_live_mobile/features/rooms/presentation/room_navigation_args.dart';
import 'package:alem_live_mobile/features/rooms/presentation/screens/room_screen.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

final appRouterProvider = Provider<GoRouter>((ref) {
  return GoRouter(
    initialLocation: LoginScreen.routePath,
    routes: [
      GoRoute(
        path: LoginScreen.routePath,
        name: LoginScreen.routeName,
        builder: (context, state) => const LoginScreen(),
      ),
      GoRoute(
        path: RegisterScreen.routePath,
        name: RegisterScreen.routeName,
        builder: (context, state) => const RegisterScreen(),
      ),
      GoRoute(
        path: HomeScreen.routePath,
        name: HomeScreen.routeName,
        builder: (context, state) => const HomeScreen(),
      ),
      GoRoute(
        path: RoomScreen.routePath,
        name: RoomScreen.routeName,
        builder: (context, state) {
          final args = state.extra is RoomNavigationArgs
              ? state.extra! as RoomNavigationArgs
              : const RoomNavigationArgs(
                  roomId: 'alemlive-demo',
                  roomName: 'AlemLive Demo',
                  isOwner: false,
                  initialMicEnabled: true,
                  initialCameraEnabled: true,
                );

          return RoomScreen(args: args);
        },
      ),
      GoRoute(
        path: ReportsScreen.routePath,
        name: ReportsScreen.routeName,
        builder: (context, state) => const ReportsScreen(),
      ),
      GoRoute(
        path: ReportDetailScreen.routePath,
        name: ReportDetailScreen.routeName,
        builder: (context, state) {
          final args = state.extra is ReportNavigationArgs
              ? state.extra! as ReportNavigationArgs
              : fallbackReportArgs();

          return ReportDetailScreen(args: args);
        },
      ),
    ],
  );
});
