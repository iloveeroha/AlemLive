import 'package:equatable/equatable.dart';

enum RecordingState { idle, recording, processing, ready, error }

class RecordingStatus extends Equatable {
  const RecordingStatus({
    required this.roomId,
    required this.state,
    this.reportId,
  });

  final String roomId;
  final RecordingState state;
  final String? reportId;

  @override
  List<Object?> get props => [roomId, state, reportId];
}
