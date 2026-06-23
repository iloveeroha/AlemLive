import 'package:alem_live_mobile/features/auth/domain/user.dart';

abstract interface class AuthRepository {
  Future<User?> restoreSession();

  Future<User> login({required String username, required String password});

  Future<User> register({required String username, required String password});

  Future<User> me();

  Future<void> logout();
}
