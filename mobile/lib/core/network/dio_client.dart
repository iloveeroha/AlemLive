import 'package:alem_live_mobile/app/config.dart';
import 'package:alem_live_mobile/core/network/network_exception.dart';
import 'package:alem_live_mobile/core/storage/secure_storage_service.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

final appConfigProvider = Provider<AppConfig>((ref) {
  return AppConfig.fromEnvironment();
});

final dioProvider = Provider<Dio>((ref) {
  final config = ref.watch(appConfigProvider);
  final storage = ref.watch(secureStorageServiceProvider);

  final dio = Dio(
    BaseOptions(
      baseUrl: config.backendBaseUrl,
      connectTimeout: const Duration(seconds: 12),
      sendTimeout: const Duration(seconds: 12),
      receiveTimeout: const Duration(seconds: 20),
      headers: {'Content-Type': 'application/json'},
    ),
  );

  dio.interceptors.add(
    InterceptorsWrapper(
      onRequest: (options, handler) async {
        final token = await storage.readAccessToken();
        if (token != null && token.isNotEmpty) {
          options.headers['Authorization'] = 'Bearer $token';
        }
        handler.next(options);
      },
      onError: (error, handler) async {
        if (error.response?.statusCode == 401) {
          await storage.clearSession();
        }
        handler.next(error);
      },
    ),
  );

  return dio;
});

NetworkException mapDioException(Object error) {
  if (error is DioException) {
    return NetworkException.fromDio(error);
  }
  if (error is NetworkException) {
    return error;
  }
  return NetworkException(message: 'Не удалось выполнить запрос', cause: error);
}
