import 'package:alem_live_mobile/app/theme.dart';
import 'package:flutter/material.dart';

class LeaveRoomDialog extends StatelessWidget {
  const LeaveRoomDialog({super.key});

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: const Text('Выйти из комнаты?'),
      content: const Text(
        'Вы отключитесь от встречи и вернетесь на главный экран.',
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(context).pop(false),
          child: const Text('Отмена'),
        ),
        FilledButton(
          onPressed: () => Navigator.of(context).pop(true),
          style: FilledButton.styleFrom(
            backgroundColor: AppTheme.blue,
            foregroundColor: Colors.white,
          ),
          child: const Text('Выйти'),
        ),
      ],
    );
  }
}
