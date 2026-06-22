import 'package:alem_live_mobile/app/theme.dart';
import 'package:alem_live_mobile/features/reports/domain/entities/action_item.dart';
import 'package:alem_live_mobile/features/reports/domain/entities/report.dart';
import 'package:flutter/material.dart';

class ActionItemsTab extends StatelessWidget {
  const ActionItemsTab({required this.report, super.key});

  final Report report;

  @override
  Widget build(BuildContext context) {
    return ListView.separated(
      padding: const EdgeInsets.all(16),
      itemCount: report.actionItems.length,
      separatorBuilder: (_, _) => const SizedBox(height: 10),
      itemBuilder: (context, index) {
        final item = report.actionItems[index];
        return Container(
          padding: const EdgeInsets.all(16),
          decoration: BoxDecoration(
            color: Colors.white,
            borderRadius: BorderRadius.circular(16),
            border: Border.all(color: AppTheme.border),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  Expanded(
                    child: Text(
                      item.task,
                      style: Theme.of(context).textTheme.titleMedium,
                    ),
                  ),
                  _StatusPill(status: item.status),
                ],
              ),
              const SizedBox(height: 12),
              _InfoLine(icon: Icons.person_outline_rounded, text: item.owner),
              const SizedBox(height: 6),
              _InfoLine(
                icon: Icons.event_outlined,
                text: item.dueDate ?? 'Срок не указан',
              ),
            ],
          ),
        );
      },
    );
  }
}

class _InfoLine extends StatelessWidget {
  const _InfoLine({required this.icon, required this.text});

  final IconData icon;
  final String text;

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Icon(icon, color: AppTheme.muted, size: 17),
        const SizedBox(width: 8),
        Expanded(child: Text(text)),
      ],
    );
  }
}

class _StatusPill extends StatelessWidget {
  const _StatusPill({required this.status});

  final ActionItemStatus status;

  @override
  Widget build(BuildContext context) {
    final color = switch (status) {
      ActionItemStatus.open => const Color(0xFFE11D48),
      ActionItemStatus.inProgress => AppTheme.blue,
      ActionItemStatus.done => const Color(0xFF039855),
    };
    final label = switch (status) {
      ActionItemStatus.open => 'Открыто',
      ActionItemStatus.inProgress => 'В работе',
      ActionItemStatus.done => 'Готово',
    };

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 9, vertical: 5),
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
