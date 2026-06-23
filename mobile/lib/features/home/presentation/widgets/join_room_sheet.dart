import 'package:alem_live_mobile/core/widgets/app_text_field.dart';
import 'package:alem_live_mobile/core/widgets/primary_button.dart';
import 'package:flutter/material.dart';

class JoinRoomSheet extends StatefulWidget {
  const JoinRoomSheet({required this.onJoin, super.key});

  final Future<void> Function(String roomName) onJoin;

  @override
  State<JoinRoomSheet> createState() => _JoinRoomSheetState();
}

class _JoinRoomSheetState extends State<JoinRoomSheet> {
  final _formKey = GlobalKey<FormState>();
  final _roomController = TextEditingController();
  bool _isSubmitting = false;

  @override
  void dispose() {
    _roomController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final bottomInset = MediaQuery.viewInsetsOf(context).bottom;

    return SafeArea(
      child: Padding(
        padding: EdgeInsets.fromLTRB(24, 0, 24, bottomInset + 24),
        child: Form(
          key: _formKey,
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Text(
                'Присоединиться',
                style: Theme.of(context).textTheme.titleLarge,
              ),
              const SizedBox(height: 18),
              AppTextField(
                controller: _roomController,
                label: 'Название комнаты',
                prefixIcon: Icons.meeting_room_outlined,
                textInputAction: TextInputAction.done,
                validator: _requiredRoomName,
              ),
              const SizedBox(height: 18),
              PrimaryButton(
                label: 'Войти',
                icon: Icons.login_rounded,
                isLoading: _isSubmitting,
                onPressed: _submit,
              ),
            ],
          ),
        ),
      ),
    );
  }

  String? _requiredRoomName(String? value) {
    if (value == null || value.trim().isEmpty) {
      return 'Введите название комнаты';
    }
    return null;
  }

  Future<void> _submit() async {
    if (!(_formKey.currentState?.validate() ?? false)) {
      return;
    }
    setState(() => _isSubmitting = true);
    try {
      await widget.onJoin(_roomController.text.trim());
    } finally {
      if (mounted) {
        setState(() => _isSubmitting = false);
      }
    }
  }
}
