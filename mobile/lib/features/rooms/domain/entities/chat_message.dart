import 'package:equatable/equatable.dart';

class ChatMessage extends Equatable {
  const ChatMessage({
    required this.id,
    required this.senderName,
    required this.text,
    required this.sentAt,
    required this.isMine,
  });

  final String id;
  final String senderName;
  final String text;
  final DateTime sentAt;
  final bool isMine;

  @override
  List<Object?> get props => [id, senderName, text, sentAt, isMine];
}
