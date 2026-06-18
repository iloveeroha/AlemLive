import 'package:alem_live_mobile/features/reports/domain/entities/action_item.dart';

class ActionItemModel extends ActionItem {
  const ActionItemModel({
    required super.task,
    required super.owner,
    required super.status,
    super.dueDate,
  });

  factory ActionItemModel.fromJson(Map<String, dynamic> json) {
    return ActionItemModel(
      task: json['task'] as String,
      owner: json['owner'] as String,
      dueDate: json['dueDate'] as String?,
      status: ActionItemStatus.values.firstWhere(
        (status) => status.name == json['status'],
        orElse: () => ActionItemStatus.open,
      ),
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
}
