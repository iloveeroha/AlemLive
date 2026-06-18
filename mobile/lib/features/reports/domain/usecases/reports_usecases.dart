import 'package:alem_live_mobile/features/reports/data/reports_repository_impl.dart';
import 'package:alem_live_mobile/features/reports/domain/entities/report.dart';
import 'package:alem_live_mobile/features/reports/domain/reports_repository.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

final reportsUseCasesProvider = Provider<ReportsUseCases>((ref) {
  return ReportsUseCases(ref.watch(reportsRepositoryProvider));
});

final reportsListProvider = FutureProvider.autoDispose<List<Report>>((ref) {
  return ref.watch(reportsUseCasesProvider).list();
});

final reportDetailProvider = FutureProvider.autoDispose.family<Report, String>((
  ref,
  reportId,
) {
  return ref.watch(reportsUseCasesProvider).detail(reportId: reportId);
});

class ReportsUseCases {
  const ReportsUseCases(this._repository);

  final ReportsRepository _repository;

  Future<List<Report>> list() {
    return _repository.list();
  }

  Future<Report> detail({required String reportId}) {
    return _repository.detail(reportId: reportId);
  }

  Future<ReportProcessingStatus> status({required String reportId}) {
    return _repository.status(reportId: reportId);
  }
}
