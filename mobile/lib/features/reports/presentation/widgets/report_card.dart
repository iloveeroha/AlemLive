import 'package:alem_live_mobile/app/theme.dart';
import 'package:alem_live_mobile/features/reports/domain/entities/report.dart';
import 'package:alem_live_mobile/features/reports/presentation/widgets/report_video_placeholder.dart';
import 'package:flutter/material.dart';

class ReportCard extends StatelessWidget {
  const ReportCard({required this.report, required this.onOpen, super.key});

  final Report report;
  final VoidCallback onOpen;

  @override
  Widget build(BuildContext context) {
    return InkWell(
      onTap: onOpen,
      borderRadius: BorderRadius.circular(18),
      child: Ink(
        padding: const EdgeInsets.all(14),
        decoration: BoxDecoration(
          color: Colors.white,
          borderRadius: BorderRadius.circular(18),
          border: Border.all(color: AppTheme.border),
          boxShadow: [
            BoxShadow(
              color: AppTheme.blue.withValues(alpha: 0.08),
              blurRadius: 18,
              offset: const Offset(0, 10),
            ),
          ],
        ),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const SizedBox(
              width: 112,
              child: ReportVideoPlaceholder(compact: true),
            ),
            const SizedBox(width: 14),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Expanded(
                        child: Text(
                          report.roomName,
                          maxLines: 2,
                          overflow: TextOverflow.ellipsis,
                          style: Theme.of(context).textTheme.titleMedium,
                        ),
                      ),
                      const SizedBox(width: 8),
                      _ReportStatusBadge(status: report.status),
                    ],
                  ),
                  const SizedBox(height: 8),
                  _MetaLine(
                    icon: Icons.calendar_today_outlined,
                    text: report.startedAtLabel,
                  ),
                  const SizedBox(height: 5),
                  _MetaLine(
                    icon: Icons.schedule_rounded,
                    text: report.durationLabel,
                  ),
                  const SizedBox(height: 12),
                  Align(
                    alignment: Alignment.centerRight,
                    child: TextButton.icon(
                      onPressed: onOpen,
                      icon: const Icon(Icons.arrow_forward_rounded, size: 18),
                      label: const Text('Открыть'),
                    ),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _MetaLine extends StatelessWidget {
  const _MetaLine({required this.icon, required this.text});

  final IconData icon;
  final String text;

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Icon(icon, color: AppTheme.muted, size: 16),
        const SizedBox(width: 6),
        Expanded(
          child: Text(
            text,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: Theme.of(context).textTheme.bodyMedium,
          ),
        ),
      ],
    );
  }
}

class _ReportStatusBadge extends StatelessWidget {
  const _ReportStatusBadge({required this.status});

  final ReportProcessingStatus status;

  @override
  Widget build(BuildContext context) {
    final color = switch (status) {
      ReportProcessingStatus.ready => const Color(0xFF039855),
      ReportProcessingStatus.processing => AppTheme.blue,
      ReportProcessingStatus.error => const Color(0xFFE11D48),
    };
    final label = switch (status) {
      ReportProcessingStatus.ready => 'Готово',
      ReportProcessingStatus.processing => 'Обработка',
      ReportProcessingStatus.error => 'Ошибка',
    };

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 5),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.12),
        borderRadius: BorderRadius.circular(999),
      ),
      child: Text(
        label,
        style: TextStyle(
          color: color,
          fontSize: 11,
          fontWeight: FontWeight.w800,
        ),
      ),
    );
  }
}
