import 'package:alem_live_mobile/features/reports/domain/entities/report.dart';
import 'package:alem_live_mobile/features/reports/presentation/widgets/transcript_tile.dart';
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
        return TranscriptTile(segment: segment, onTimecodeTap: onTimecodeTap);
      },
    );
  }
}
