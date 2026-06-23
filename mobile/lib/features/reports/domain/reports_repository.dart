import 'package:alem_live_mobile/features/reports/domain/entities/report.dart';

abstract interface class ReportsRepository {
  Future<List<Report>> list();

  Future<Report> detail({required String reportId});

  Future<ReportProcessingStatus> status({required String reportId});

  String videoPath({required String reportId});
}
