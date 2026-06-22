import 'package:alem_live_mobile/app/theme.dart';
import 'package:flutter/material.dart';

class GridBackground extends StatelessWidget {
  const GridBackground({
    required this.child,
    this.padding = EdgeInsets.zero,
    super.key,
  });

  final Widget child;
  final EdgeInsetsGeometry padding;

  @override
  Widget build(BuildContext context) {
    return CustomPaint(
      painter: const _GridPainter(),
      child: Padding(padding: padding, child: child),
    );
  }
}

class _GridPainter extends CustomPainter {
  const _GridPainter();

  @override
  void paint(Canvas canvas, Size size) {
    const step = 28.0;
    final paint = Paint()
      ..color = AppTheme.gridLine
      ..strokeWidth = 0.8;

    for (double x = 0; x <= size.width; x += step) {
      canvas.drawLine(Offset(x, 0), Offset(x, size.height), paint);
    }
    for (double y = 0; y <= size.height; y += step) {
      canvas.drawLine(Offset(0, y), Offset(size.width, y), paint);
    }
  }

  @override
  bool shouldRepaint(covariant CustomPainter oldDelegate) => false;
}
