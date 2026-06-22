import 'package:alem_live_mobile/app/theme.dart';
import 'package:flutter/material.dart';

class LoadingView extends StatelessWidget {
  const LoadingView({this.message = 'Загрузка...', super.key});

  final String message;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          const CircularProgressIndicator(color: AppTheme.blue),
          const SizedBox(height: 14),
          Text(message),
        ],
      ),
    );
  }
}
