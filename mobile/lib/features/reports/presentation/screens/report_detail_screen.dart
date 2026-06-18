import 'package:alem_live_mobile/app/theme.dart';
import 'package:alem_live_mobile/core/widgets/error_view.dart';
import 'package:alem_live_mobile/core/widgets/loading_view.dart';
import 'package:alem_live_mobile/features/reports/data/models/report_model.dart';
import 'package:alem_live_mobile/features/reports/domain/entities/report.dart';
import 'package:alem_live_mobile/features/reports/domain/usecases/reports_usecases.dart';
import 'package:alem_live_mobile/features/reports/presentation/widgets/action_items_tab.dart';
import 'package:alem_live_mobile/features/reports/presentation/widgets/activity_tab.dart';
import 'package:alem_live_mobile/features/reports/presentation/widgets/ai_question_tab.dart';
import 'package:alem_live_mobile/features/reports/presentation/widgets/notes_tab.dart';
import 'package:alem_live_mobile/features/reports/presentation/widgets/report_tabs.dart';
import 'package:alem_live_mobile/features/reports/presentation/widgets/report_video_placeholder.dart';
import 'package:alem_live_mobile/features/reports/presentation/widgets/transcript_tab.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class ReportNavigationArgs {
  const ReportNavigationArgs({required this.report});

  final Report report;
}

class ReportDetailScreen extends ConsumerStatefulWidget {
  const ReportDetailScreen({required this.args, super.key});

  static const routeName = 'report-detail';
  static const routePath = '/reports/detail';

  final ReportNavigationArgs args;

  @override
  ConsumerState<ReportDetailScreen> createState() => _ReportDetailScreenState();
}

class _ReportDetailScreenState extends ConsumerState<ReportDetailScreen> {
  int _selectedTab = 0;

  @override
  Widget build(BuildContext context) {
    final reportAsync = ref.watch(reportDetailProvider(widget.args.report.id));

    return Scaffold(
      backgroundColor: const Color(0xFFF8FAFF),
      appBar: AppBar(title: const Text('AI-отчет')),
      body: reportAsync.when(
        loading: () => const LoadingView(message: 'Загружаем отчет...'),
        error: (error, _) => ErrorView(
          message: error.toString(),
          onRetry: () =>
              ref.invalidate(reportDetailProvider(widget.args.report.id)),
        ),
        data: _buildReport,
      ),
    );
  }

  Widget _buildReport(Report report) {
    return Column(
      children: [
        Expanded(
          child: CustomScrollView(
            slivers: [
              SliverToBoxAdapter(
                child: Padding(
                  padding: const EdgeInsets.fromLTRB(16, 8, 16, 12),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      const ReportVideoPlaceholder(),
                      const SizedBox(height: 14),
                      Row(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Expanded(
                            child: Column(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                Text(
                                  report.roomName,
                                  style: Theme.of(
                                    context,
                                  ).textTheme.headlineMedium,
                                ),
                                const SizedBox(height: 8),
                                Text(
                                  '${report.startedAtLabel} • ${report.durationLabel}',
                                  style: Theme.of(context).textTheme.bodyMedium,
                                ),
                              ],
                            ),
                          ),
                          _StatusBadge(status: report.status),
                        ],
                      ),
                    ],
                  ),
                ),
              ),
              SliverPersistentHeader(
                pinned: true,
                delegate: _TabsHeaderDelegate(
                  child: ColoredBox(
                    color: const Color(0xFFF8FAFF),
                    child: Padding(
                      padding: const EdgeInsets.symmetric(vertical: 8),
                      child: ReportTabs(
                        selectedIndex: _selectedTab,
                        onChanged: (index) {
                          setState(() => _selectedTab = index);
                        },
                      ),
                    ),
                  ),
                ),
              ),
              SliverFillRemaining(
                child: _SelectedTabContent(
                  selectedTab: _selectedTab,
                  report: report,
                ),
              ),
            ],
          ),
        ),
      ],
    );
  }
}

class _SelectedTabContent extends StatelessWidget {
  const _SelectedTabContent({required this.selectedTab, required this.report});

  final int selectedTab;
  final Report report;

  @override
  Widget build(BuildContext context) {
    return switch (selectedTab) {
      0 => NotesTab(report: report),
      1 => ActionItemsTab(report: report),
      2 => ActivityTab(report: report),
      3 => TranscriptTab(report: report),
      _ => AiQuestionTab(report: report),
    };
  }
}

class _StatusBadge extends StatelessWidget {
  const _StatusBadge({required this.status});

  final ReportProcessingStatus status;

  @override
  Widget build(BuildContext context) {
    final color = switch (status) {
      ReportProcessingStatus.ready => const Color(0xFF039855),
      ReportProcessingStatus.processing => AppTheme.blue,
      ReportProcessingStatus.error => const Color(0xFFE11D48),
    };

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 7),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.12),
        borderRadius: BorderRadius.circular(999),
      ),
      child: Text(
        status.statusLabel,
        style: TextStyle(
          color: color,
          fontWeight: FontWeight.w800,
          fontSize: 12,
        ),
      ),
    );
  }
}

class _TabsHeaderDelegate extends SliverPersistentHeaderDelegate {
  const _TabsHeaderDelegate({required this.child});

  final Widget child;

  @override
  double get minExtent => 62;

  @override
  double get maxExtent => 62;

  @override
  Widget build(
    BuildContext context,
    double shrinkOffset,
    bool overlapsContent,
  ) {
    return child;
  }

  @override
  bool shouldRebuild(covariant _TabsHeaderDelegate oldDelegate) {
    return oldDelegate.child != child;
  }
}

ReportNavigationArgs fallbackReportArgs() {
  return ReportNavigationArgs(report: ReportModel.mockReports().first);
}

extension _ReportStatusLabel on ReportProcessingStatus {
  String get statusLabel {
    return switch (this) {
      ReportProcessingStatus.processing => 'Обрабатывается',
      ReportProcessingStatus.ready => 'Готово',
      ReportProcessingStatus.error => 'Ошибка',
    };
  }
}
