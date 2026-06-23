import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

final secureStorageServiceProvider = Provider<SecureStorageService>((ref) {
  return const SecureStorageService();
});

class SecureStorageService {
  const SecureStorageService([this._storage = const FlutterSecureStorage()]);

  static const _accessTokenKey = 'access_token';
  static const _refreshTokenKey = 'refresh_token';
  static const _idTokenKey = 'id_token';
  static const _expiresAtKey = 'expires_at';
  static const _usernameKey = 'username';

  final FlutterSecureStorage _storage;

  Future<void> saveSession({
    required String accessToken,
    required String username,
    String? refreshToken,
    String? idToken,
    DateTime? expiresAt,
  }) async {
    await Future.wait([
      _storage.write(key: _accessTokenKey, value: accessToken),
      _storage.write(key: _usernameKey, value: username),
      if (refreshToken != null)
        _storage.write(key: _refreshTokenKey, value: refreshToken),
      if (idToken != null) _storage.write(key: _idTokenKey, value: idToken),
      if (expiresAt != null)
        _storage.write(key: _expiresAtKey, value: expiresAt.toIso8601String()),
    ]);
  }

  Future<String?> readAccessToken() {
    return _storage.read(key: _accessTokenKey);
  }

  Future<String?> readRefreshToken() {
    return _storage.read(key: _refreshTokenKey);
  }

  Future<String?> readIdToken() {
    return _storage.read(key: _idTokenKey);
  }

  Future<DateTime?> readExpiresAt() async {
    final value = await _storage.read(key: _expiresAtKey);
    if (value == null) {
      return null;
    }
    return DateTime.tryParse(value);
  }

  Future<String?> readUsername() {
    return _storage.read(key: _usernameKey);
  }

  Future<void> clearSession() async {
    await Future.wait([
      _storage.delete(key: _accessTokenKey),
      _storage.delete(key: _refreshTokenKey),
      _storage.delete(key: _idTokenKey),
      _storage.delete(key: _expiresAtKey),
      _storage.delete(key: _usernameKey),
    ]);
  }
}
