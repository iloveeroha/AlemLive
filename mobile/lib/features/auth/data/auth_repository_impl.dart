import 'dart:convert';

import 'package:alem_live_mobile/app/config.dart';
import 'package:alem_live_mobile/core/errors/app_exception.dart';
import 'package:alem_live_mobile/core/network/dio_client.dart';
import 'package:alem_live_mobile/core/storage/secure_storage_service.dart';
import 'package:alem_live_mobile/features/auth/data/auth_api_client.dart';
import 'package:alem_live_mobile/features/auth/data/keycloak_auth_service.dart';
import 'package:alem_live_mobile/features/auth/domain/auth_repository.dart';
import 'package:alem_live_mobile/features/auth/domain/auth_token.dart';
import 'package:alem_live_mobile/features/auth/domain/user.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

final authRepositoryProvider = Provider<AuthRepository>((ref) {
  return AuthRepositoryImpl(
    apiClient: ref.watch(authApiClientProvider),
    keycloakAuthService: ref.watch(keycloakAuthServiceProvider),
    storage: ref.watch(secureStorageServiceProvider),
    config: ref.watch(appConfigProvider),
  );
});

class AuthRepositoryImpl implements AuthRepository {
  const AuthRepositoryImpl({
    required this.apiClient,
    required this.keycloakAuthService,
    required this.storage,
    required this.config,
  });

  final AuthApiClient apiClient;
  final KeycloakAuthService keycloakAuthService;
  final SecureStorageService storage;
  final AppConfig config;

  @override
  Future<User?> restoreSession() async {
    final token = await storage.readAccessToken();
    if (token == null || token.trim().isEmpty) {
      return null;
    }

    try {
      await _refreshStoredTokenIfNeeded(token);
      return await me();
    } catch (_) {
      await storage.clearSession();
      return null;
    }
  }

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
      await _saveSession(token: token, user: user);
      return user;
    } catch (error) {
      if (!config.enableMockFallback) {
        throw mapDioException(error);
      }
      return _mockLogin(username: username);
    }
  }

  @override
  Future<User> loginWithKeycloak() async {
    try {
      final response = await keycloakAuthService.login();
      final token = _tokenFromResponse(response);
      final user = _userFromResponse(response);
      await _saveSession(token: token, user: user);
      return await me();
    } catch (error) {
      if (error is AppException) {
        rethrow;
      }
      throw mapDioException(error);
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
      await _saveSession(token: token, user: user);
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

  Future<void> _saveSession({
    required AuthToken token,
    required User user,
  }) async {
    await storage.saveSession(
      accessToken: token.accessToken,
      refreshToken: token.refreshToken,
      idToken: token.idToken,
      expiresAt: token.expiresAt,
      username: user.username,
    );
  }

  Future<void> _refreshStoredTokenIfNeeded(String accessToken) async {
    final expiresAt =
        await storage.readExpiresAt() ?? _expiresAtFromToken(accessToken);
    if (expiresAt == null ||
        expiresAt.isAfter(DateTime.now().add(const Duration(seconds: 30)))) {
      return;
    }

    final refreshToken = await storage.readRefreshToken();
    if (refreshToken == null || refreshToken.isEmpty) {
      return;
    }

    final response = await apiClient.refreshToken(refreshToken: refreshToken);
    final token = _tokenFromResponse(response);
    final user = _userFromResponse(
      response,
      fallbackUsername: await storage.readUsername() ?? 'user',
    );
    await _saveSession(token: token, user: user);
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
    if (accessToken == null || accessToken.toString().trim().isEmpty) {
      throw const AppException('Keycloak не вернул access token');
    }
    final expiresInSeconds = int.tryParse(
      response['expires_in']?.toString() ?? '',
    );
    final expiresAt =
        _expiresAtFromToken(accessToken.toString()) ??
        _expiresAtFromJson(response['expiresAt']) ??
        DateTime.now().add(Duration(seconds: expiresInSeconds ?? 3600));
    return AuthToken(
      accessToken: accessToken.toString(),
      refreshToken: response['refresh_token']?.toString(),
      idToken: response['id_token']?.toString(),
      expiresAt: expiresAt,
    );
  }

  User _userFromResponse(
    Map<String, dynamic> response, {
    String fallbackUsername = 'user',
  }) {
    final user = response['user'] is Map<String, dynamic>
        ? response['user'] as Map<String, dynamic>
        : response;
    final token =
        response['accessToken'] ??
        response['access_token'] ??
        response['token'];
    if (response['user'] is! Map<String, dynamic> && token != null) {
      return _userFromToken(token.toString(), fallbackUsername);
    }
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

  User _userFromToken(String token, String fallbackUsername) {
    final claims = _claimsFromToken(token);
    final username =
        claims['preferred_username']?.toString() ??
        claims['email']?.toString() ??
        claims['name']?.toString() ??
        fallbackUsername.trim();
    final displayName =
        claims['name']?.toString() ??
        claims['preferred_username']?.toString() ??
        username;
    return User(
      id: claims['sub']?.toString() ?? username.toLowerCase(),
      username: username,
      displayName: displayName,
    );
  }

  DateTime? _expiresAtFromJson(Object? value) {
    if (value == null) {
      return null;
    }
    return DateTime.tryParse(value.toString());
  }

  DateTime? _expiresAtFromToken(String token) {
    final exp = _claimsFromToken(token)['exp'];
    if (exp is num) {
      return DateTime.fromMillisecondsSinceEpoch(exp.toInt() * 1000);
    }
    return null;
  }

  Map<String, dynamic> _claimsFromToken(String token) {
    try {
      final parts = token.split('.');
      if (parts.length != 3) {
        return <String, dynamic>{};
      }
      final payload = utf8.decode(
        base64Url.decode(base64Url.normalize(parts[1])),
      );
      final decoded = jsonDecode(payload);
      if (decoded is Map<String, dynamic>) {
        return decoded;
      }
    } catch (_) {
      return <String, dynamic>{};
    }
    return <String, dynamic>{};
  }
}
