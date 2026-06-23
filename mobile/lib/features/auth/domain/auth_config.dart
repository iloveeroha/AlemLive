class AuthConfig {
  const AuthConfig({
    required this.enabled,
    required this.clientId,
    required this.authorizationEndpoint,
    required this.logoutEndpoint,
  });

  factory AuthConfig.fromJson(Map<String, dynamic> json) {
    return AuthConfig(
      enabled: json['enabled'] == true,
      clientId: json['clientId']?.toString() ?? '',
      authorizationEndpoint: _uriFromJson(json['authorizationEndpoint']),
      logoutEndpoint: _uriFromJson(json['logoutEndpoint']),
    );
  }

  static Uri? _uriFromJson(Object? value) {
    final text = value?.toString().trim() ?? '';
    if (text.isEmpty) {
      return null;
    }
    return Uri.tryParse(text);
  }

  final bool enabled;
  final String clientId;
  final Uri? authorizationEndpoint;
  final Uri? logoutEndpoint;

  bool get canStartKeycloakLogin {
    return enabled && clientId.isNotEmpty && authorizationEndpoint != null;
  }
}
