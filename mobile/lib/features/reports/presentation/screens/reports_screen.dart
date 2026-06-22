import 'package:alem_live_mobile/core/widgets/error_view.dart';
import 'package:alem_live_mobile/core/widgets/grid_background.dart';
import 'package:alem_live_mobile/core/widgets/loading_view.dart';
import 'package:alem_live_mobile/features/home/presentation/home_screen.dart';
import 'package:alem_live_mobile/features/reports/domain/entities/report.dart';
import 'package:alem_live_mobile/features/reports/domain/usecases/reports_usecases.dart';
import 'package:alem_live_mobile/features/reports/presentation/screens/report_detail_screen.dart';
import 'package:alem_live_mobile/features/reports/presentation/widgets/report_card.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

class ReportsScreen extends ConsumerWidget {
  const ReportsScreen({super.key});

  static const routeName = 'reports';
  static const routePath = '/reports';

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final reportsAsync = ref.watch(reportsListProvider);

    return Scaffold(
      appBar: AppBar(
        title: const Text('Отчеты'),
        actions: [
          IconButton(
            tooltip: 'Главное меню',
            onPressed: () => context.go(HomeScreen.routePath),
            icon: const Icon(Icons.home_rounded),
          ),
          const SizedBox(width: 8),
        ],
      ),
      body: GridBackground(
        child: reportsAsync.when(
          loading: () => const LoadingView(message: 'Загружаем отчеты...'),
          error: (error, _) => ErrorView(
            message: error.toString(),
            onRetry: () => ref.invalidate(reportsListProvider),
          ),
          data: (reports) {
            if (reports.isEmpty) {
              return const Center(child: Text('Отчетов пока нет'));
            }

            return RefreshIndicator(
              onRefresh: () async {
                ref.invalidate(reportsListProvider);
                await ref.read(reportsListProvider.future);
              },
              child: ListView.separated(
                padding: const EdgeInsets.fromLTRB(16, 8, 16, 24),
                itemCount: reports.length,
                separatorBuilder: (_, _) => const SizedBox(height: 12),
                itemBuilder: (context, index) {
                  final report = reports[index];
                  return ReportCard(
                    report: report,
                    onOpen: () => _openReport(context, report),
                  );
                },
              ),
            );
          },
        ),
      ),
    );
  }

  void _openReport(BuildContext context, Report report) {
    context.goNamed(
      ReportDetailScreen.routeName,
      extra: ReportNavigationArgs(report: report),
    );
  }
}
