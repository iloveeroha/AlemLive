import 'package:alem_live_mobile/app/theme.dart';
import 'package:alem_live_mobile/features/reports/domain/entities/transcript_segment.dart';
import 'package:flutter/material.dart';

class TranscriptTile extends StatelessWidget {
  const TranscriptTile({
    required this.segment,
    required this.onTimecodeTap,
    super.key,
  });

  final TranscriptSegment segment;
  final ValueChanged<String>? onTimecodeTap;

  @override
  Widget build(BuildContext context) {
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
  }
}
