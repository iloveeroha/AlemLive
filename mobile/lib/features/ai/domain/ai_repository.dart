abstract interface class AiRepository {
  Future<String> askQuestion({
    required String reportId,
    required String roomName,
    required String question,
    required String fallbackTakeaway,
  });
}
