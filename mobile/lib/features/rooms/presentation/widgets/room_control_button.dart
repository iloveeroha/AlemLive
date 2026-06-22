import 'package:alem_live_mobile/app/theme.dart';
import 'package:flutter/material.dart';

class RoomControlButton extends StatelessWidget {
  const RoomControlButton({
    required this.icon,
    required this.label,
    required this.onPressed,
    this.backgroundColor = Colors.black,
    this.iconSize = 20,
    super.key,
  });

  final IconData icon;
  final String label;
  final VoidCallback onPressed;
  final Color backgroundColor;
  final double iconSize;

  @override
  Widget build(BuildContext context) {
    final accentColor = backgroundColor;

    return Tooltip(
      message: label,
      child: InkWell(
        onTap: onPressed,
        borderRadius: BorderRadius.circular(14),
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 2, vertical: 2),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Container(
                width: 40,
                height: 40,
                decoration: BoxDecoration(
                  color: accentColor.withValues(alpha: 0.1),
                  shape: BoxShape.circle,
                  border: Border.all(
                    color: accentColor.withValues(alpha: 0.24),
                  ),
                ),
                child: Icon(icon, color: accentColor, size: iconSize),
              ),
              const SizedBox(height: 4),
              Text(
                label,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                textAlign: TextAlign.center,
                style: TextStyle(
                  color: accentColor == Colors.black
                      ? AppTheme.ink
                      : accentColor,
                  fontSize: 9,
                  fontWeight: FontWeight.w800,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
