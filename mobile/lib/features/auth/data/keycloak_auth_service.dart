import 'dart:async';
import 'dart:convert';
import 'dart:math';

import 'package:alem_live_mobile/core/errors/app_exception.dart';
import 'package:alem_live_mobile/features/auth/data/auth_api_client.dart';
import 'package:alem_live_mobile/features/auth/domain/auth_config.dart';
import 'package:app_links/app_links.dart';
import 'package:crypto/crypto.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:url_launcher/url_launcher.dart';

final keycloakAuthServiceProvider = Provider<KeycloakAuthService>((ref) {
  return KeycloakAuthService(
    apiClient: ref.watch(authApiClientProvider),
    appLinks: AppLinks(),
  );
});

class KeycloakAuthService {
  const KeycloakAuthService({required this.apiClient, required this.appLinks});

  static const redirectUri = 'alemlive://auth/callback';
  static const _callbackScheme = 'alemlive';
  static const _callbackHost = 'auth';
  static const _callbackPath = '/callback';

  final AuthApiClient apiClient;
  final AppLinks appLinks;

  Future<Map<String, dynamic>> login() async {
    final config = AuthConfig.fromJson(await apiClient.authConfig());
    if (!config.canStartKeycloakLogin) {
      throw const AppException('Keycloak is not configured');
    }

    final verifier = _randomPKCEValue();
    final state = _randomPKCEValue(24);
    final challenge = _sha256Base64Url(verifier);
    final callbackFuture = _waitForCallback(state);

    final authUrl = config.authorizationEndpoint!.replace(
      queryParameters: <String, String>{
        ...config.authorizationEndpoint!.queryParameters,
        'client_id': config.clientId,
        'response_type': 'code',
        'scope': 'openid profile email',
        'redirect_uri': redirectUri,
        'code_challenge': challenge,
        'code_challenge_method': 'S256',
        'state': state,
      },
    );

    final launched = await launchUrl(
      authUrl,
      mode: LaunchMode.externalApplication,
    );
    if (!launched) {
      throw const AppException('Could not open Keycloak');
    }

    final callback = await callbackFuture.timeout(
      const Duration(minutes: 3),
      onTimeout: () {
        throw const AppException('Keycloak login was cancelled or timed out');
      },
    );
    final error =
        callback.queryParameters['error_description'] ??
        callback.queryParameters['error'];
    if (error != null && error.isNotEmpty) {
      throw AppException(error);
    }

    final code = callback.queryParameters['code'];
    if (code == null || code.isEmpty) {
      throw const AppException('Keycloak did not return an authorization code');
    }

    return apiClient.exchangeAuthCode(
      code: code,
      redirectUri: redirectUri,
      codeVerifier: verifier,
    );
  }

  Future<Uri> _waitForCallback(String expectedState) async {
    final completer = Completer<Uri>();
    StreamSubscription<Uri>? subscription;

    void completeIfAuthCallback(Uri uri) {
      if (!_isCallback(uri)) {
        return;
      }
      final state = uri.queryParameters['state'];
      if (state != expectedState) {
        return;
      }
      if (!completer.isCompleted) {
        completer.complete(uri);
      }
    }

    subscription = appLinks.uriLinkStream.listen(
      completeIfAuthCallback,
      onError: (Object error) {
        if (!completer.isCompleted) {
          completer.completeError(
            AppException('Could not receive Keycloak callback: $error'),
          );
        }
      },
    );

    final initialLink = await appLinks.getInitialLink();
    if (initialLink != null) {
      completeIfAuthCallback(initialLink);
    }

    return completer.future.whenComplete(() => subscription?.cancel());
  }

  bool _isCallback(Uri uri) {
    return uri.scheme == _callbackScheme &&
        uri.host == _callbackHost &&
        uri.path == _callbackPath;
  }

  String _randomPKCEValue([int length = 64]) {
    final random = Random.secure();
    final bytes = List<int>.generate(length, (_) => random.nextInt(256));
    return base64UrlEncode(bytes).replaceAll('=', '');
  }

  String _sha256Base64Url(String value) {
    final digest = sha256.convert(utf8.encode(value));
    return base64UrlEncode(digest.bytes).replaceAll('=', '');
  }
}
