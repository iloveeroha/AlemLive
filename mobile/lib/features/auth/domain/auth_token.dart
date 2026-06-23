import 'package:equatable/equatable.dart';

class AuthToken extends Equatable {
  const AuthToken({
    required this.accessToken,
    required this.expiresAt,
    this.refreshToken,
    this.idToken,
  });

  final String accessToken;
  final DateTime expiresAt;
  final String? refreshToken;
  final String? idToken;

  @override
  List<Object?> get props => [accessToken, expiresAt, refreshToken, idToken];
}
