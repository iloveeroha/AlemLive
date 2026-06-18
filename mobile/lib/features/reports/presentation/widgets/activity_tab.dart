import 'package:alem_live_mobile/app/theme.dart';
import 'package:alem_live_mobile/features/reports/domain/entities/report.dart';
import 'package:flutter/material.dart';

class ActivityTab extends StatelessWidget {
  const ActivityTab({required this.report, super.key});

  final Report report;

  @override
  Widget build(BuildContext context) {
    if (report.speakerActivity.isEmpty) {
      return const Center(child: Text('Активность пока недоступна'));
    }

    final mostActive = report.speakerActivity.firstWhere(
      (activity) => activity.isMostActive,
      orElse: () => report.speakerActivity.first,
    );

    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        Container(
          padding: const EdgeInsets.all(16),
          decoration: BoxDecoration(
            color: AppTheme.blue,
            borderRadius: BorderRadius.circular(16),
          ),
          child: Row(
            children: [
              const Icon(Icons.workspace_premium_rounded, color: Colors.white),
              const SizedBox(width: 10),
              Expanded(
                child: Text(
                  'Самый активный: ${mostActive.speakerName}',
                  style: const TextStyle(
                    color: Colors.white,
                    fontWeight: FontWeight.w800,
                    fontSize: 16,
                  ),
                ),
              ),
            ],
          ),
        ),
        const SizedBox(height: 14),
        ...report.speakerActivity.map((activity) {
          return Container(
            margin: const EdgeInsets.only(bottom: 10),
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
                        activity.speakerName,
                        style: Theme.of(context).textTheme.titleMedium,
                      ),
                    ),
                    Text(
                      activity.talkTimeLabel,
                      style: const TextStyle(fontWeight: FontWeight.w800),
                    ),
                  ],
                ),
                const SizedBox(height: 10),
                LinearProgressIndicator(
                  value: activity.participationPercent / 100,
                  minHeight: 8,
                  borderRadius: BorderRadius.circular(999),
                  backgroundColor: const Color(0xFFEAF0FF),
                  color: activity.isMostActive
                      ? AppTheme.blue
                      : AppTheme.blueDark,
                ),
                const SizedBox(height: 8),
                Text('${activity.participationPercent}% участия'),
              ],
            ),
          );
        }),
      ],
    );
  }
}
