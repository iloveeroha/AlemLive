import 'package:alem_live_mobile/core/errors/app_exception.dart';
import 'package:dio/dio.dart';

class NetworkException extends AppException {
  const NetworkException({
    required String message,
    this.statusCode,
    Object? cause,
  }) : super(message, cause: cause);

  final int? statusCode;

  bool get isUnauthorized => statusCode == 401;
  bool get isNotFound => statusCode == 404;
  bool get isTimeout => statusCode == null;

  factory NetworkException.fromDio(DioException error) {
    final statusCode = error.response?.statusCode;

    if (error.type == DioExceptionType.connectionTimeout ||
        error.type == DioExceptionType.sendTimeout ||
        error.type == DioExceptionType.receiveTimeout) {
      return NetworkException(
        message: 'Сервер не ответил вовремя',
        cause: error,
      );
    }

    if (error.type == DioExceptionType.connectionError) {
      return NetworkException(message: 'Backend недоступен', cause: error);
    }

    final responseData = error.response?.data;
    final responseMessage = responseData is Map<String, dynamic>
        ? responseData['message'] ?? responseData['error']
        : null;

    return NetworkException(
      message: responseMessage?.toString() ?? _messageForStatus(statusCode),
      statusCode: statusCode,
      cause: error,
    );
  }

  static String _messageForStatus(int? statusCode) {
    if (statusCode == 400) {
      return 'Некорректный запрос';
    }
    if (statusCode == 401) {
      return 'Сессия истекла. Войдите снова';
    }
    if (statusCode == 403) {
      return 'Недостаточно прав';
    }
    if (statusCode == 404) {
      return 'Endpoint не найден';
    }
    if (statusCode != null && statusCode >= 500) {
      return 'Ошибка сервера';
    }
    return 'Не удалось выполнить запрос';
  }
}
