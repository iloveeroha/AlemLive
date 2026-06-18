import 'package:alem_live_mobile/core/widgets/app_text_field.dart';
import 'package:alem_live_mobile/core/widgets/primary_button.dart';
import 'package:alem_live_mobile/core/widgets/secondary_button.dart';
import 'package:alem_live_mobile/features/auth/presentation/auth_controller.dart';
import 'package:alem_live_mobile/features/auth/presentation/widgets/auth_shell.dart';
import 'package:alem_live_mobile/features/home/presentation/home_screen.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

class RegisterScreen extends ConsumerStatefulWidget {
  const RegisterScreen({super.key});

  static const routeName = 'register';
  static const routePath = '/register';

  @override
  ConsumerState<RegisterScreen> createState() => _RegisterScreenState();
}

class _RegisterScreenState extends ConsumerState<RegisterScreen> {
  final _formKey = GlobalKey<FormState>();
  final _usernameController = TextEditingController();
  final _passwordController = TextEditingController();
  final _repeatPasswordController = TextEditingController();

  @override
  void dispose() {
    _usernameController.dispose();
    _passwordController.dispose();
    _repeatPasswordController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final authState = ref.watch(authControllerProvider);

    ref.listen(authControllerProvider, (previous, next) {
      if (next.isAuthenticated) {
        context.go(HomeScreen.routePath);
      }
      if (next.errorMessage != null) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text(next.errorMessage!)));
      }
    });

    return AuthShell(
      title: 'Создать аккаунт',
      subtitle: 'Регистрация сохранит токен и откроет главный экран.',
      child: Form(
        key: _formKey,
        child: Column(
          children: [
            AppTextField(
              controller: _usernameController,
              label: 'Имя пользователя',
              prefixIcon: Icons.person_outline_rounded,
              textInputAction: TextInputAction.next,
              validator: _required,
            ),
            const SizedBox(height: 14),
            AppTextField(
              controller: _passwordController,
              label: 'Пароль',
              prefixIcon: Icons.lock_outline_rounded,
              obscureText: true,
              textInputAction: TextInputAction.next,
              validator: _passwordValidator,
            ),
            const SizedBox(height: 14),
            AppTextField(
              controller: _repeatPasswordController,
              label: 'Повтор пароля',
              prefixIcon: Icons.lock_reset_rounded,
              obscureText: true,
              textInputAction: TextInputAction.done,
              validator: _repeatPasswordValidator,
            ),
            const SizedBox(height: 22),
            PrimaryButton(
              label: 'Зарегистрироваться',
              icon: Icons.person_add_alt_1_rounded,
              isLoading: authState.isLoading,
              onPressed: _submit,
            ),
            const SizedBox(height: 12),
            SecondaryButton(
              label: 'Уже есть аккаунт',
              icon: Icons.arrow_back_rounded,
              onPressed: () => context.go('/'),
            ),
          ],
        ),
      ),
    );
  }

  String? _required(String? value) {
    if (value == null || value.trim().isEmpty) {
      return 'Заполните поле';
    }
    return null;
  }

  String? _passwordValidator(String? value) {
    if (value == null || value.length < 6) {
      return 'Минимум 6 символов';
    }
    return null;
  }

  String? _repeatPasswordValidator(String? value) {
    if (value != _passwordController.text) {
      return 'Пароли не совпадают';
    }
    return null;
  }

  void _submit() {
    if (!(_formKey.currentState?.validate() ?? false)) {
      return;
    }
    ref
        .read(authControllerProvider.notifier)
        .register(
          username: _usernameController.text,
          password: _passwordController.text,
        );
  }
}
