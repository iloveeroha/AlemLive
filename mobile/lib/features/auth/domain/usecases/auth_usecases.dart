import 'package:alem_live_mobile/features/auth/data/auth_repository_impl.dart';
import 'package:alem_live_mobile/features/auth/domain/auth_repository.dart';
import 'package:alem_live_mobile/features/auth/domain/user.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

final loginUseCaseProvider = Provider<LoginUseCase>((ref) {
  return LoginUseCase(ref.watch(authRepositoryProvider));
});

final restoreSessionUseCaseProvider = Provider<RestoreSessionUseCase>((ref) {
  return RestoreSessionUseCase(ref.watch(authRepositoryProvider));
});

final loginWithKeycloakUseCaseProvider = Provider<LoginWithKeycloakUseCase>((
  ref,
) {
  return LoginWithKeycloakUseCase(ref.watch(authRepositoryProvider));
});

final registerUseCaseProvider = Provider<RegisterUseCase>((ref) {
  return RegisterUseCase(ref.watch(authRepositoryProvider));
});

final logoutUseCaseProvider = Provider<LogoutUseCase>((ref) {
  return LogoutUseCase(ref.watch(authRepositoryProvider));
});

final meUseCaseProvider = Provider<MeUseCase>((ref) {
  return MeUseCase(ref.watch(authRepositoryProvider));
});

class LoginUseCase {
  const LoginUseCase(this._repository);

  final AuthRepository _repository;

  Future<User> call({required String username, required String password}) {
    return _repository.login(username: username, password: password);
  }
}

class RestoreSessionUseCase {
  const RestoreSessionUseCase(this._repository);

  final AuthRepository _repository;

  Future<User?> call() {
    return _repository.restoreSession();
  }
}

class LoginWithKeycloakUseCase {
  const LoginWithKeycloakUseCase(this._repository);

  final AuthRepository _repository;

  Future<User> call() {
    return _repository.loginWithKeycloak();
  }
}

class RegisterUseCase {
  const RegisterUseCase(this._repository);

  final AuthRepository _repository;

  Future<User> call({required String username, required String password}) {
    return _repository.register(username: username, password: password);
  }
}

class LogoutUseCase {
  const LogoutUseCase(this._repository);

  final AuthRepository _repository;

  Future<void> call() {
    return _repository.logout();
  }
}

class MeUseCase {
  const MeUseCase(this._repository);

  final AuthRepository _repository;

  Future<User> call() {
    return _repository.me();
  }
}
