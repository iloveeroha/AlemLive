import 'package:alem_live_mobile/app/theme.dart';
import 'package:flutter/material.dart';

class ReportTabs extends StatelessWidget {
  const ReportTabs({
    required this.selectedIndex,
    required this.onChanged,
    super.key,
  });

  final int selectedIndex;
  final ValueChanged<int> onChanged;

  static const labels = [
    'Заметки',
    'Действия',
    'Активность',
    'Транскрипт',
    'Вопрос AI',
  ];

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      height: 46,
      child: ListView.separated(
        scrollDirection: Axis.horizontal,
        padding: const EdgeInsets.symmetric(horizontal: 16),
        itemCount: labels.length,
        separatorBuilder: (_, _) => const SizedBox(width: 8),
        itemBuilder: (context, index) {
          final selected = selectedIndex == index;
          return ChoiceChip(
            selected: selected,
            label: Text(labels[index]),
            onSelected: (_) => onChanged(index),
            selectedColor: AppTheme.blue,
            backgroundColor: Colors.white,
            labelStyle: TextStyle(
              color: selected ? Colors.white : AppTheme.ink,
              fontWeight: FontWeight.w800,
            ),
            side: BorderSide(color: selected ? AppTheme.blue : AppTheme.border),
          );
        },
      ),
    );
  }
}
