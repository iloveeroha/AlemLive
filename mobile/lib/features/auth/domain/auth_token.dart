import 'package:equatable/equatable.dart';

class AuthToken extends Equatable {
  const AuthToken({required this.accessToken, required this.expiresAt});

  final String accessToken;
  final DateTime expiresAt;

  @override
  List<Object?> get props => [accessToken, expiresAt];
}
