package httpapi

import (
	"context"
	"crypto"
	"crypto/hmac"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type authUser struct {
	ID       string
	Name     string
	Email    string
	Username string
}

type authContextKey struct{}

type jwkSet struct {
	Keys []jwkKey `json:"keys"`
}

type jwkKey struct {
	Kid string   `json:"kid"`
	Kty string   `json:"kty"`
	Alg string   `json:"alg"`
	Use string   `json:"use"`
	N   string   `json:"n"`
	E   string   `json:"e"`
	X5C []string `json:"x5c"`
}

type jwtHeader struct {
	Alg string `json:"alg"`
	Kid string `json:"kid"`
	Typ string `json:"typ"`
}

type jwtClaims struct {
	Subject           string          `json:"sub"`
	Issuer            string          `json:"iss"`
	Audience          json.RawMessage `json:"aud"`
	AuthorizedParty   string          `json:"azp"`
	ExpiresAt         int64           `json:"exp"`
	NotBefore         int64           `json:"nbf"`
	IssuedAt          int64           `json:"iat"`
	Name              string          `json:"name"`
	PreferredUsername string          `json:"preferred_username"`
	Email             string          `json:"email"`
}

type authCredentialsRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type localAuthClaims struct {
	Subject           string `json:"sub"`
	Name              string `json:"name"`
	PreferredUsername string `json:"preferred_username"`
	Email             string `json:"email,omitempty"`
	ExpiresAt         int64  `json:"exp"`
	IssuedAt          int64  `json:"iat"`
}

type jwksCache struct {
	mu        sync.Mutex
	keys      map[string]*rsa.PublicKey
	expiresAt time.Time
}

func (s *Server) authEnabled() bool {
	return s.cfg.KeycloakIssuerURL != "" && s.cfg.KeycloakClientID != ""
}

func (s *Server) withAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if (!s.authEnabled() && !s.cfg.LocalAuthEnabled) || r.Method == http.MethodOptions || isPublicPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		user, err := s.authenticateRequest(r)
		if err != nil {
			writeError(w, http.StatusUnauthorized, err.Error())
			return
		}

		ctx := context.WithValue(r.Context(), authContextKey{}, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func isPublicPath(path string) bool {
	if strings.HasPrefix(path, "/api/reports/") && strings.HasSuffix(path, "/recording/stream") {
		return true
	}
	return path == "/healthz" ||
		path == "/api/config" ||
		path == "/api/diagnostics/recording" ||
		path == "/api/auth/config" ||
		path == "/api/auth/token" ||
		path == "/api/auth/login" ||
		path == "/api/auth/register" ||
		path == "/api/auth/logout" ||
		path == "/api/livekit/webhook"
}

func (s *Server) authLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}

	credentials, ok := readAuthCredentials(w, r)
	if !ok {
		return
	}

	if s.authEnabled() {
		if response, err := s.loginWithKeycloak(r, credentials); err == nil {
			writeJSON(w, http.StatusOK, response)
			return
		} else if !s.cfg.LocalAuthEnabled {
			writeError(w, http.StatusUnauthorized, "Invalid username or password")
			return
		}
	}

	if !s.cfg.LocalAuthEnabled {
		writeError(w, http.StatusServiceUnavailable, "Local auth is disabled")
		return
	}

	response, err := s.localAuthResponse(credentials.Username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not create local auth token")
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) authRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}

	credentials, ok := readAuthCredentials(w, r)
	if !ok {
		return
	}
	if !s.cfg.LocalAuthEnabled {
		writeError(w, http.StatusServiceUnavailable, "Registration is handled by the identity provider")
		return
	}

	response, err := s.localAuthResponse(credentials.Username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not create local auth token")
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) authLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) authMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}

	user, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Missing user")
		return
	}

	writeJSON(w, http.StatusOK, authUserPayload(user))
}

func readAuthCredentials(w http.ResponseWriter, r *http.Request) (authCredentialsRequest, bool) {
	var req authCredentialsRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16*1024)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return authCredentialsRequest{}, false
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || strings.TrimSpace(req.Password) == "" {
		writeError(w, http.StatusBadRequest, "username and password are required")
		return authCredentialsRequest{}, false
	}
	return req, true
}

func (s *Server) loginWithKeycloak(r *http.Request, credentials authCredentialsRequest) (map[string]any, error) {
	tokenURL := s.cfg.KeycloakTokenURL
	if tokenURL == "" {
		tokenURL = authEndpoint(s.cfg.KeycloakIssuerURL, "token")
	}

	form := url.Values{}
	form.Set("grant_type", "password")
	form.Set("client_id", s.cfg.KeycloakClientID)
	form.Set("username", credentials.Username)
	form.Set("password", credentials.Password)
	form.Set("scope", "openid profile email")

	request, err := http.NewRequestWithContext(r.Context(), http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 1<<20))
		return nil, errors.New("keycloak login failed")
	}

	var tokenPayload map[string]any
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&tokenPayload); err != nil {
		return nil, err
	}
	accessToken, _ := tokenPayload["access_token"].(string)
	if accessToken == "" {
		return nil, errors.New("keycloak response did not include access token")
	}

	user := userFromAccessToken(accessToken, credentials.Username)
	return map[string]any{
		"accessToken": accessToken,
		"token":       accessToken,
		"expiresIn":   tokenPayload["expires_in"],
		"user":        authUserPayload(user),
	}, nil
}

func (s *Server) localAuthResponse(username string) (map[string]any, error) {
	now := s.clock().UTC()
	user := authUser{
		ID:       localUserID(username),
		Name:     username,
		Username: username,
	}
	token, expiresAt, err := s.createLocalAccessToken(user, now)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"accessToken": token,
		"token":       token,
		"expiresAt":   expiresAt.Format(time.RFC3339),
		"user":        authUserPayload(user),
	}, nil
}

func authUserPayload(user authUser) map[string]string {
	return map[string]string{
		"id":          user.ID,
		"name":        firstNonEmpty(user.Name, user.Username, user.ID),
		"username":    firstNonEmpty(user.Username, user.Name, user.ID),
		"displayName": firstNonEmpty(user.Name, user.Username, user.ID),
		"email":       user.Email,
	}
}

func userFromAccessToken(accessToken string, fallbackUsername string) authUser {
	var claims jwtClaims
	parts := strings.Split(accessToken, ".")
	if len(parts) == 3 {
		_ = decodeJWTPart(parts[1], &claims)
	}
	name := firstNonEmpty(claims.Name, claims.PreferredUsername, claims.Email, fallbackUsername)
	username := firstNonEmpty(claims.PreferredUsername, fallbackUsername, claims.Email, claims.Subject)
	return authUser{
		ID:       firstNonEmpty(claims.Subject, localUserID(username)),
		Name:     name,
		Email:    claims.Email,
		Username: username,
	}
}

func (s *Server) authToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	if !s.authEnabled() {
		writeError(w, http.StatusServiceUnavailable, "Keycloak is not configured")
		return
	}

	var req struct {
		Code         string `json:"code"`
		RedirectURI  string `json:"redirectUri"`
		CodeVerifier string `json:"codeVerifier"`
		RefreshToken string `json:"refreshToken"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16*1024)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	hasRefreshToken := strings.TrimSpace(req.RefreshToken) != ""
	hasCodeExchange := strings.TrimSpace(req.Code) != "" && strings.TrimSpace(req.RedirectURI) != "" && strings.TrimSpace(req.CodeVerifier) != ""
	if !hasRefreshToken && !hasCodeExchange {
		writeError(w, http.StatusBadRequest, "code, redirectUri and codeVerifier are required")
		return
	}

	tokenURL := s.cfg.KeycloakTokenURL
	if tokenURL == "" {
		tokenURL = authEndpoint(s.cfg.KeycloakIssuerURL, "token")
	}

	form := url.Values{}
	form.Set("client_id", s.cfg.KeycloakClientID)
	if hasRefreshToken {
		form.Set("grant_type", "refresh_token")
		form.Set("refresh_token", req.RefreshToken)
	} else {
		form.Set("grant_type", "authorization_code")
		form.Set("code", req.Code)
		form.Set("redirect_uri", req.RedirectURI)
		form.Set("code_verifier", req.CodeVerifier)
	}

	request, err := http.NewRequestWithContext(r.Context(), http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not create token request")
		return
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		writeError(w, http.StatusBadGateway, "Could not exchange Keycloak code")
		return
	}
	defer response.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(response.StatusCode)
	_, _ = io.Copy(w, io.LimitReader(response.Body, 1<<20))
}

func (s *Server) authenticateRequest(r *http.Request) (authUser, error) {
	const prefix = "Bearer "
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	token := ""
	if strings.HasPrefix(header, prefix) {
		token = strings.TrimSpace(strings.TrimPrefix(header, prefix))
	}
	if token == "" {
		token = strings.TrimSpace(r.URL.Query().Get("token"))
	}
	if token == "" {
		return authUser{}, errors.New("Missing bearer token")
	}

	if s.cfg.LocalAuthEnabled {
		if user, err := s.validateLocalAccessToken(token); err == nil {
			return user, nil
		}
	}

	return s.validateAccessToken(token)
}

func (s *Server) createLocalAccessToken(user authUser, now time.Time) (string, time.Time, error) {
	secret := strings.TrimSpace(s.cfg.LocalAuthSecret)
	if secret == "" {
		return "", time.Time{}, errors.New("local auth secret is empty")
	}

	expiresAt := now.Add(s.cfg.TokenTTL)
	if s.cfg.TokenTTL <= 0 {
		expiresAt = now.Add(12 * time.Hour)
	}
	claims := localAuthClaims{
		Subject:           firstNonEmpty(user.ID, localUserID(user.Username)),
		Name:              firstNonEmpty(user.Name, user.Username, user.ID),
		PreferredUsername: firstNonEmpty(user.Username, user.Name, user.ID),
		Email:             user.Email,
		ExpiresAt:         expiresAt.Unix(),
		IssuedAt:          now.Unix(),
	}

	payload, err := json.Marshal(claims)
	if err != nil {
		return "", time.Time{}, err
	}
	payloadPart := base64.RawURLEncoding.EncodeToString(payload)
	signature := signLocalAuth(payloadPart, secret)
	return "local." + payloadPart + "." + signature, expiresAt, nil
}

func (s *Server) validateLocalAccessToken(token string) (authUser, error) {
	secret := strings.TrimSpace(s.cfg.LocalAuthSecret)
	if secret == "" || !strings.HasPrefix(token, "local.") {
		return authUser{}, errors.New("not a local auth token")
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return authUser{}, errors.New("invalid local auth token")
	}
	expectedSignature := signLocalAuth(parts[1], secret)
	if !hmac.Equal([]byte(parts[2]), []byte(expectedSignature)) {
		return authUser{}, errors.New("invalid local auth signature")
	}

	rawClaims, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return authUser{}, errors.New("invalid local auth payload")
	}
	var claims localAuthClaims
	if err := json.Unmarshal(rawClaims, &claims); err != nil {
		return authUser{}, errors.New("invalid local auth claims")
	}
	if claims.ExpiresAt <= s.clock().Unix() {
		return authUser{}, errors.New("local auth token expired")
	}

	username := firstNonEmpty(claims.PreferredUsername, claims.Name, claims.Subject)
	return authUser{
		ID:       firstNonEmpty(claims.Subject, localUserID(username)),
		Name:     firstNonEmpty(claims.Name, username),
		Email:    claims.Email,
		Username: username,
	}, nil
}

func signLocalAuth(payloadPart, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payloadPart))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func localUserID(username string) string {
	username = strings.TrimSpace(strings.ToLower(username))
	if username == "" {
		return "local-user"
	}
	var b strings.Builder
	lastDash := false
	for _, r := range username {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			lastDash = false
		case !lastDash:
			b.WriteRune('-')
			lastDash = true
		}
	}
	id := strings.Trim(b.String(), "-")
	if id == "" {
		return "local-user"
	}
	return "local-" + id
}

func (s *Server) validateAccessToken(token string) (authUser, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return authUser{}, errors.New("Invalid bearer token")
	}

	var header jwtHeader
	if err := decodeJWTPart(parts[0], &header); err != nil {
		return authUser{}, errors.New("Invalid token header")
	}
	if header.Alg != "RS256" || header.Kid == "" {
		return authUser{}, errors.New("Unsupported token signature")
	}

	var claims jwtClaims
	if err := decodeJWTPart(parts[1], &claims); err != nil {
		return authUser{}, errors.New("Invalid token claims")
	}
	if err := s.validateClaims(claims); err != nil {
		return authUser{}, err
	}

	publicKey, err := s.publicKeyForKID(header.Kid)
	if err != nil {
		return authUser{}, err
	}

	signed := []byte(parts[0] + "." + parts[1])
	digest := sha256.Sum256(signed)
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return authUser{}, errors.New("Invalid token signature")
	}
	if err := rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, digest[:], signature); err != nil {
		return authUser{}, errors.New("Invalid token signature")
	}

	name := firstNonEmpty(claims.Name, claims.PreferredUsername, claims.Email, claims.Subject)
	return authUser{
		ID:       claims.Subject,
		Name:     name,
		Email:    claims.Email,
		Username: claims.PreferredUsername,
	}, nil
}

func (s *Server) validateClaims(claims jwtClaims) error {
	now := s.clock().Unix()
	if claims.Issuer != s.cfg.KeycloakIssuerURL {
		return errors.New("Invalid token issuer")
	}
	if claims.ExpiresAt == 0 || claims.ExpiresAt <= now {
		return errors.New("Bearer token expired")
	}
	if claims.NotBefore > 0 && claims.NotBefore > now {
		return errors.New("Bearer token is not active yet")
	}
	if !claims.hasAudience(s.cfg.KeycloakClientID) && claims.AuthorizedParty != s.cfg.KeycloakClientID {
		return errors.New("Invalid token audience")
	}
	return nil
}

func (claims jwtClaims) hasAudience(clientID string) bool {
	var single string
	if err := json.Unmarshal(claims.Audience, &single); err == nil {
		return single == clientID
	}

	var multiple []string
	if err := json.Unmarshal(claims.Audience, &multiple); err == nil {
		for _, audience := range multiple {
			if audience == clientID {
				return true
			}
		}
	}

	return false
}

func (s *Server) publicKeyForKID(kid string) (*rsa.PublicKey, error) {
	s.jwks.mu.Lock()
	defer s.jwks.mu.Unlock()

	if s.jwks.keys != nil && s.clock().Before(s.jwks.expiresAt) {
		if key := s.jwks.keys[kid]; key != nil {
			return key, nil
		}
	}

	keys, err := s.fetchJWKS()
	if err != nil {
		return nil, err
	}
	s.jwks.keys = keys
	s.jwks.expiresAt = s.clock().Add(10 * time.Minute)

	if key := keys[kid]; key != nil {
		return key, nil
	}
	return nil, errors.New("Token signing key not found")
}

func (s *Server) fetchJWKS() (map[string]*rsa.PublicKey, error) {
	jwksURL := s.cfg.KeycloakJWKSURL
	if jwksURL == "" {
		jwksURL = s.cfg.KeycloakIssuerURL + "/protocol/openid-connect/certs"
	}

	request, err := http.NewRequest(http.MethodGet, jwksURL, nil)
	if err != nil {
		return nil, errors.New("Could not create JWKS request")
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, errors.New("Could not load Keycloak JWKS")
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, errors.New("Could not load Keycloak JWKS")
	}

	var set jwkSet
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&set); err != nil {
		return nil, errors.New("Invalid Keycloak JWKS")
	}

	keys := make(map[string]*rsa.PublicKey, len(set.Keys))
	for _, key := range set.Keys {
		publicKey, err := key.publicKey()
		if err == nil && key.Kid != "" {
			keys[key.Kid] = publicKey
		}
	}
	if len(keys) == 0 {
		return nil, errors.New("Keycloak JWKS has no usable keys")
	}
	return keys, nil
}

func (key jwkKey) publicKey() (*rsa.PublicKey, error) {
	if len(key.X5C) > 0 {
		certDER, err := base64.StdEncoding.DecodeString(key.X5C[0])
		if err == nil {
			cert, err := x509.ParseCertificate(certDER)
			if err == nil {
				if publicKey, ok := cert.PublicKey.(*rsa.PublicKey); ok {
					return publicKey, nil
				}
			}
		}
	}

	if key.Kty != "RSA" || key.N == "" || key.E == "" {
		return nil, errors.New("Unsupported JWK key")
	}

	nBytes, err := base64.RawURLEncoding.DecodeString(key.N)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(key.E)
	if err != nil {
		return nil, err
	}

	exponent := 0
	for _, b := range eBytes {
		exponent = exponent<<8 + int(b)
	}
	if exponent == 0 {
		return nil, errors.New("Invalid JWK exponent")
	}

	return &rsa.PublicKey{N: new(big.Int).SetBytes(nBytes), E: exponent}, nil
}

func decodeJWTPart(part string, target any) error {
	raw, err := base64.RawURLEncoding.DecodeString(part)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, target)
}

func userFromContext(ctx context.Context) (authUser, bool) {
	user, ok := ctx.Value(authContextKey{}).(authUser)
	return user, ok
}
