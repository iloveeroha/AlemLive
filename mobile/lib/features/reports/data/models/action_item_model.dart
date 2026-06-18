import 'package:alem_live_mobile/features/reports/domain/entities/action_item.dart';

class ActionItemModel extends ActionItem {
  const ActionItemModel({
    required super.task,
    required super.owner,
    required super.status,
    super.dueDate,
  });

  factory ActionItemModel.fromJson(Map<String, dynamic> json) {
    final rawStatus = (json['status'] ?? '').toString().toLowerCase();
    return ActionItemModel(
      task: (json['task'] ?? json['title'] ?? json['description'] ?? '')
          .toString(),
      owner: (json['owner'] ?? 'Команда').toString(),
      dueDate: (json['dueDate'] ?? json['due'])?.toString(),
      status: _parseStatus(rawStatus),
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'task': task,
      'owner': owner,
      'dueDate': dueDate,
      'status': status.name,
    };
  }

  static ActionItemStatus _parseStatus(String value) {
    return switch (value) {
      'done' || 'completed' || 'complete' || 'closed' => ActionItemStatus.done,
      'inprogress' ||
      'in_progress' ||
      'in-progress' ||
      'processing' => ActionItemStatus.inProgress,
      _ => ActionItemStatus.open,
    };
  }
}
