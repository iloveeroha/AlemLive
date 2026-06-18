import 'package:equatable/equatable.dart';

enum ActionItemStatus { open, inProgress, done }

class ActionItem extends Equatable {
  const ActionItem({
    required this.task,
    required this.owner,
    required this.status,
    this.dueDate,
  });

  final String task;
  final String owner;
  final String? dueDate;
  final ActionItemStatus status;

  String get statusLabel {
    return switch (status) {
      ActionItemStatus.open => 'Открыто',
      ActionItemStatus.inProgress => 'В работе',
      ActionItemStatus.done => 'Готово',
    };
  }

  @override
  List<Object?> get props => [task, owner, dueDate, status];
}
