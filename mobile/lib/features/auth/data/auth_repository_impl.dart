import 'package:alem_live_mobile/app/config.dart';
import 'package:alem_live_mobile/core/errors/app_exception.dart';
import 'package:alem_live_mobile/core/network/dio_client.dart';
import 'package:alem_live_mobile/core/storage/secure_storage_service.dart';
import 'package:alem_live_mobile/features/auth/data/auth_api_client.dart';
import 'package:alem_live_mobile/features/auth/domain/auth_repository.dart';
import 'package:alem_live_mobile/features/auth/domain/auth_token.dart';
import 'package:alem_live_mobile/features/auth/domain/user.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

final authRepositoryProvider = Provider<AuthRepository>((ref) {
  return AuthRepositoryImpl(
    apiClient: ref.watch(authApiClientProvider),
    storage: ref.watch(secureStorageServiceProvider),
    config: ref.watch(appConfigProvider),
  );
});

class AuthRepositoryImpl implements AuthRepository {
  const AuthRepositoryImpl({
    required this.apiClient,
    required this.storage,
    required this.config,
  });

  final AuthApiClient apiClient;
  final SecureStorageService storage;
  final AppConfig config;

  @override
  Future<User> login({
    required String username,
    required String password,
  }) async {
    _validateCredentials(username: username, password: password);

    try {
      final response = await apiClient.login(
        username: username.trim(),
        password: password,
      );
      final token = _tokenFromResponse(response);
      final user = _userFromResponse(response, fallbackUsername: username);
      await storage.saveSession(
        accessToken: token.accessToken,
        username: user.username,
      );
      return user;
    } catch (error) {
      if (!config.enableMockFallback) {
        throw mapDioException(error);
      }
      return _mockLogin(username: username);
    }
  }

  @override
  Future<User> register({
    required String username,
    required String password,
  }) async {
    _validateCredentials(username: username, password: password);

    try {
      final response = await apiClient.register(
        username: username.trim(),
        password: password,
      );
      final token = _tokenFromResponse(response);
      final user = _userFromResponse(response, fallbackUsername: username);
      await storage.saveSession(
        accessToken: token.accessToken,
        username: user.username,
      );
      return user;
    } catch (error) {
      if (!config.enableMockFallback) {
        throw mapDioException(error);
      }
      return _mockLogin(username: username);
    }
  }

  @override
  Future<User> me() async {
    try {
      final response = await apiClient.me();
      return _userFromResponse(response);
    } catch (error) {
      if (!config.enableMockFallback) {
        throw mapDioException(error);
      }
      final username = await storage.readUsername();
      return User(
        id: (username ?? 'demo-user').toLowerCase(),
        username: username ?? 'demo',
        displayName: username ?? 'Demo User',
      );
    }
  }

  @override
  Future<void> logout() async {
    try {
      await apiClient.logout();
    } catch (error) {
      if (!config.enableMockFallback) {
        throw mapDioException(error);
      }
    } finally {
      await storage.clearSession();
    }
  }

  Future<User> _mockLogin({required String username}) async {
    final token = AuthToken(
      accessToken: 'dev-token-${DateTime.now().millisecondsSinceEpoch}',
      expiresAt: DateTime.now().add(const Duration(hours: 12)),
    );

    await storage.saveSession(
      accessToken: token.accessToken,
      username: username.trim(),
    );

    return User(
      id: username.trim().toLowerCase(),
      username: username.trim(),
      displayName: username.trim(),
    );
  }

  void _validateCredentials({
    required String username,
    required String password,
  }) {
    if (username.trim().isEmpty) {
      throw const AppException('Введите имя пользователя');
    }
    if (password.length < 6) {
      throw const AppException('Пароль должен быть не короче 6 символов');
    }
  }

  AuthToken _tokenFromResponse(Map<String, dynamic> response) {
    final accessToken =
        response['accessToken'] ??
        response['access_token'] ??
        response['token'];
    return AuthToken(
      accessToken: accessToken?.toString() ?? 'missing-token',
      expiresAt: DateTime.now().add(const Duration(hours: 12)),
    );
  }

  User _userFromResponse(
    Map<String, dynamic> response, {
    String fallbackUsername = 'user',
  }) {
    final user = response['user'] is Map<String, dynamic>
        ? response['user'] as Map<String, dynamic>
        : response;
    final username =
        user['username']?.toString() ??
        user['name']?.toString() ??
        fallbackUsername.trim();
    return User(
      id: user['id']?.toString() ?? username.toLowerCase(),
      username: username,
      displayName:
          user['displayName']?.toString() ??
          user['name']?.toString() ??
          username,
    );
  }
}
