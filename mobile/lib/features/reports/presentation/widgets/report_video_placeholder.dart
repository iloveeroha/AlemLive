import 'package:alem_live_mobile/app/theme.dart';
import 'package:flutter/material.dart';

class ReportVideoPlaceholder extends StatelessWidget {
  const ReportVideoPlaceholder({this.compact = false, super.key});

  final bool compact;

  @override
  Widget build(BuildContext context) {
    // TODO: Replace this placeholder with a real video player.
    return AspectRatio(
      aspectRatio: compact ? 16 / 10 : 16 / 9,
      child: Container(
        decoration: BoxDecoration(
          color: const Color(0xFF101828),
          borderRadius: BorderRadius.circular(compact ? 12 : 18),
        ),
        child: Stack(
          children: [
            Positioned.fill(
              child: DecoratedBox(
                decoration: BoxDecoration(
                  borderRadius: BorderRadius.circular(compact ? 12 : 18),
                  gradient: LinearGradient(
                    begin: Alignment.topLeft,
                    end: Alignment.bottomRight,
                    colors: [
                      AppTheme.blue.withValues(alpha: 0.55),
                      const Color(0xFF101828),
                    ],
                  ),
                ),
              ),
            ),
            Center(
              child: Container(
                width: compact ? 38 : 58,
                height: compact ? 38 : 58,
                decoration: BoxDecoration(
                  color: Colors.white.withValues(alpha: 0.18),
                  shape: BoxShape.circle,
                ),
                child: Icon(
                  Icons.play_arrow_rounded,
                  color: Colors.white,
                  size: compact ? 28 : 42,
                ),
              ),
            ),
            if (!compact)
              const Positioned(
                left: 16,
                bottom: 14,
                child: Text(
                  'Video preview',
                  style: TextStyle(
                    color: Colors.white,
                    fontWeight: FontWeight.w800,
                  ),
                ),
              ),
          ],
        ),
      ),
    );
  }
}
