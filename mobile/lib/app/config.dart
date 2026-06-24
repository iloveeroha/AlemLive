class AppConfig {
  const AppConfig({
    required this.backendBaseUrl,
    required this.liveKitUrl,
    required this.enableMockFallback,
  });

  factory AppConfig.fromEnvironment() {
    return const AppConfig(
      backendBaseUrl: String.fromEnvironment(
        'BACKEND_BASE_URL',
        defaultValue: 'http://localhost:8088',
      ),
      liveKitUrl: String.fromEnvironment(
        'LIVEKIT_URL',
        defaultValue: 'ws://localhost:7880',
      ),
      enableMockFallback: bool.fromEnvironment(
        'ENABLE_MOCK_FALLBACK',
        defaultValue: false,
      ),
    );
  }

  final String backendBaseUrl;
  final String liveKitUrl;
  final bool enableMockFallback;
}
