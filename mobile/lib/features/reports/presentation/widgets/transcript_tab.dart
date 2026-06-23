import 'package:alem_live_mobile/app/theme.dart';
import 'package:alem_live_mobile/features/reports/domain/entities/report.dart';
import 'package:flutter/material.dart';

class TranscriptTab extends StatelessWidget {
  const TranscriptTab({required this.report, this.onTimecodeTap, super.key});

  final Report report;
  final ValueChanged<String>? onTimecodeTap;

  @override
  Widget build(BuildContext context) {
    if (report.transcript.isEmpty) {
      return const Center(child: Text('Транскрипт пока недоступен'));
    }

    return ListView.separated(
      padding: const EdgeInsets.all(16),
      itemCount: report.transcript.length,
      separatorBuilder: (_, _) => const SizedBox(height: 10),
      itemBuilder: (context, index) {
        final segment = report.transcript[index];
        return Container(
          padding: const EdgeInsets.all(14),
          decoration: BoxDecoration(
            color: Colors.white,
            borderRadius: BorderRadius.circular(16),
            border: Border.all(color: AppTheme.border),
          ),
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              TextButton(
                onPressed: () => onTimecodeTap?.call(segment.timecode),
                child: Text(segment.timecode),
              ),
              const SizedBox(width: 8),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      segment.speakerName,
                      style: Theme.of(context).textTheme.titleMedium,
                    ),
                    const SizedBox(height: 5),
                    Text(segment.text),
                  ],
                ),
              ),
            ],
          ),
        );
      },
    );
  }
}
