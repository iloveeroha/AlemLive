import 'package:alem_live_mobile/core/network/dio_client.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

final authApiClientProvider = Provider<AuthApiClient>((ref) {
  return AuthApiClient(ref.watch(dioProvider));
});

class AuthApiClient {
  const AuthApiClient(this._dio);

  final Dio _dio;

  Future<Map<String, dynamic>> login({
    required String username,
    required String password,
  }) async {
    final response = await _dio.post<Map<String, dynamic>>(
      '/api/auth/login',
      data: {'username': username, 'password': password},
    );
    return response.data ?? <String, dynamic>{};
  }

  Future<Map<String, dynamic>> register({
    required String username,
    required String password,
  }) async {
    final response = await _dio.post<Map<String, dynamic>>(
      '/api/auth/register',
      data: {'username': username, 'password': password},
    );
    return response.data ?? <String, dynamic>{};
  }

  Future<Map<String, dynamic>> authConfig() async {
    final response = await _dio.get<Map<String, dynamic>>('/api/auth/config');
    return response.data ?? <String, dynamic>{};
  }

  Future<Map<String, dynamic>> exchangeAuthCode({
    required String code,
    required String redirectUri,
    required String codeVerifier,
  }) async {
    final response = await _dio.post<Map<String, dynamic>>(
      '/api/auth/token',
      data: {
        'code': code,
        'redirectUri': redirectUri,
        'codeVerifier': codeVerifier,
      },
    );
    return response.data ?? <String, dynamic>{};
  }

  Future<void> logout() async {
    await _dio.post<void>('/api/auth/logout');
  }

  Future<Map<String, dynamic>> me() async {
    final response = await _dio.get<Map<String, dynamic>>('/api/auth/me');
    return response.data ?? <String, dynamic>{};
  }
}
