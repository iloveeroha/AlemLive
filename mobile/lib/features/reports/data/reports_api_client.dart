import 'package:alem_live_mobile/core/network/dio_client.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

final reportsApiClientProvider = Provider<ReportsApiClient>((ref) {
  return ReportsApiClient(ref.watch(dioProvider));
});

class ReportsApiClient {
  const ReportsApiClient(this._dio);

  final Dio _dio;

  Future<List<dynamic>> list() async {
    final response = await _dio.get<dynamic>('/api/reports');
    final data = response.data;
    if (data is List<dynamic>) {
      return data;
    }
    if (data is Map<String, dynamic> && data['reports'] is List<dynamic>) {
      return data['reports'] as List<dynamic>;
    }
    return const [];
  }

  Future<Map<String, dynamic>> detail({required String reportId}) async {
    final response = await _dio.get<Map<String, dynamic>>(
      '/api/reports/$reportId',
    );
    return response.data ?? <String, dynamic>{};
  }

  Future<Map<String, dynamic>> status({required String reportId}) async {
    final response = await _dio.get<Map<String, dynamic>>(
      '/api/reports/$reportId/status',
    );
    return response.data ?? <String, dynamic>{};
  }
}
