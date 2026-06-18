import 'package:alem_live_mobile/app/config.dart';
import 'package:alem_live_mobile/core/network/dio_client.dart';
import 'package:alem_live_mobile/features/reports/data/models/report_model.dart';
import 'package:alem_live_mobile/features/reports/data/reports_api_client.dart';
import 'package:alem_live_mobile/features/reports/domain/entities/report.dart';
import 'package:alem_live_mobile/features/reports/domain/reports_repository.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

final reportsRepositoryProvider = Provider<ReportsRepository>((ref) {
  return ReportsRepositoryImpl(
    apiClient: ref.watch(reportsApiClientProvider),
    config: ref.watch(appConfigProvider),
  );
});

class ReportsRepositoryImpl implements ReportsRepository {
  const ReportsRepositoryImpl({required this.apiClient, required this.config});

  final ReportsApiClient apiClient;
  final AppConfig config;

  @override
  Future<List<Report>> list() async {
    try {
      final response = await apiClient.list();
      return response
          .whereType<Map<String, dynamic>>()
          .map(ReportModel.fromJson)
          .toList();
    } catch (error) {
      if (!config.enableMockFallback) {
        throw mapDioException(error);
      }
      return ReportModel.mockReports();
    }
  }

  @override
  Future<Report> detail({required String reportId}) async {
    try {
      return ReportModel.fromJson(await apiClient.detail(reportId: reportId));
    } catch (error) {
      if (!config.enableMockFallback) {
        throw mapDioException(error);
      }
      return ReportModel.mockReports().firstWhere(
        (report) => report.id == reportId,
        orElse: () => ReportModel.mockReports().first,
      );
    }
  }

  @override
  Future<ReportProcessingStatus> status({required String reportId}) async {
    try {
      final response = await apiClient.status(reportId: reportId);
      final rawStatus = response['status']?.toString();
      return ReportProcessingStatus.values.firstWhere(
        (status) => status.name == rawStatus,
        orElse: () => ReportProcessingStatus.processing,
      );
    } catch (error) {
      if (!config.enableMockFallback) {
        throw mapDioException(error);
      }
      return (await detail(reportId: reportId)).status;
    }
  }
}
