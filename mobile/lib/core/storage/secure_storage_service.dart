import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

final secureStorageServiceProvider = Provider<SecureStorageService>((ref) {
  return const SecureStorageService();
});

class SecureStorageService {
  const SecureStorageService([this._storage = const FlutterSecureStorage()]);

  static const _accessTokenKey = 'access_token';
  static const _usernameKey = 'username';

  final FlutterSecureStorage _storage;

  Future<void> saveSession({
    required String accessToken,
    required String username,
  }) async {
    await Future.wait([
      _storage.write(key: _accessTokenKey, value: accessToken),
      _storage.write(key: _usernameKey, value: username),
    ]);
  }

  Future<String?> readAccessToken() {
    return _storage.read(key: _accessTokenKey);
  }

  Future<String?> readUsername() {
    return _storage.read(key: _usernameKey);
  }

  Future<void> clearSession() async {
    await Future.wait([
      _storage.delete(key: _accessTokenKey),
      _storage.delete(key: _usernameKey),
    ]);
  }
}
