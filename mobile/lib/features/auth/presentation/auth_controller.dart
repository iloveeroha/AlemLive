import 'package:alem_live_mobile/core/errors/app_exception.dart';
import 'package:alem_live_mobile/features/auth/domain/usecases/auth_usecases.dart';
import 'package:alem_live_mobile/features/auth/domain/user.dart';
import 'package:equatable/equatable.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

final authControllerProvider = NotifierProvider<AuthController, AuthState>(
  AuthController.new,
);

enum AuthStatus { idle, loading, authenticated, failure }

class AuthState extends Equatable {
  const AuthState({required this.status, this.user, this.errorMessage});

  const AuthState.idle() : this(status: AuthStatus.idle);

  final AuthStatus status;
  final User? user;
  final String? errorMessage;

  bool get isLoading => status == AuthStatus.loading;
  bool get isAuthenticated => status == AuthStatus.authenticated;

  AuthState copyWith({AuthStatus? status, User? user, String? errorMessage}) {
    return AuthState(
      status: status ?? this.status,
      user: user ?? this.user,
      errorMessage: errorMessage,
    );
  }

  @override
  List<Object?> get props => [status, user, errorMessage];
}

class AuthController extends Notifier<AuthState> {
  @override
  AuthState build() {
    return const AuthState.idle();
  }

  Future<void> login({
    required String username,
    required String password,
  }) async {
    await _runAuthAction(
      () => ref
          .read(loginUseCaseProvider)
          .call(username: username, password: password),
    );
  }

  Future<void> register({
    required String username,
    required String password,
  }) async {
    await _runAuthAction(
      () => ref
          .read(registerUseCaseProvider)
          .call(username: username, password: password),
    );
  }

  Future<void> logout() async {
    await ref.read(logoutUseCaseProvider).call();
    state = const AuthState.idle();
  }

  Future<void> _runAuthAction(Future<User> Function() action) async {
    state = state.copyWith(status: AuthStatus.loading, errorMessage: null);
    try {
      final user = await action();
      state = AuthState(status: AuthStatus.authenticated, user: user);
    } on AppException catch (error) {
      state = state.copyWith(
        status: AuthStatus.failure,
        errorMessage: error.message,
      );
    } catch (_) {
      state = state.copyWith(
        status: AuthStatus.failure,
        errorMessage: 'Не удалось выполнить авторизацию',
      );
    }
  }
}
