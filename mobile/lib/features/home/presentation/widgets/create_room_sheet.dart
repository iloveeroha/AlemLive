import 'package:alem_live_mobile/core/widgets/app_text_field.dart';
import 'package:alem_live_mobile/core/widgets/primary_button.dart';
import 'package:flutter/material.dart';

class CreateRoomSheet extends StatefulWidget {
  const CreateRoomSheet({required this.onCreate, super.key});

  final Future<void> Function(
    String roomName,
    bool initialMicEnabled,
    bool initialCameraEnabled,
  )
  onCreate;

  @override
  State<CreateRoomSheet> createState() => _CreateRoomSheetState();
}

class _CreateRoomSheetState extends State<CreateRoomSheet> {
  final _formKey = GlobalKey<FormState>();
  final _roomController = TextEditingController();
  bool _microphoneEnabled = true;
  bool _cameraEnabled = true;
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
                'Создать комнату',
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
              SwitchListTile(
                contentPadding: EdgeInsets.zero,
                value: _microphoneEnabled,
                onChanged: _isSubmitting
                    ? null
                    : (value) {
                        setState(() => _microphoneEnabled = value);
                      },
                title: const Text('Микрофон'),
                secondary: const Icon(Icons.mic_rounded),
              ),
              SwitchListTile(
                contentPadding: EdgeInsets.zero,
                value: _cameraEnabled,
                onChanged: _isSubmitting
                    ? null
                    : (value) {
                        setState(() => _cameraEnabled = value);
                      },
                title: const Text('Камера'),
                secondary: const Icon(Icons.videocam_rounded),
              ),
              const SizedBox(height: 18),
              PrimaryButton(
                label: 'Создать',
                icon: Icons.add_rounded,
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
      await widget.onCreate(
        _roomController.text.trim(),
        _microphoneEnabled,
        _cameraEnabled,
      );
    } finally {
      if (mounted) {
        setState(() => _isSubmitting = false);
      }
    }
  }
}
