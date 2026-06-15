package httpapi

import (
	"context"
	"crypto"
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
		if !s.authEnabled() || r.Method == http.MethodOptions || isPublicPath(r.URL.Path) {
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
	return path == "/healthz" || path == "/api/config" || path == "/api/auth/config" || path == "/api/auth/token" || path == "/api/livekit/webhook"
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
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16*1024)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if strings.TrimSpace(req.Code) == "" || strings.TrimSpace(req.RedirectURI) == "" || strings.TrimSpace(req.CodeVerifier) == "" {
		writeError(w, http.StatusBadRequest, "code, redirectUri and codeVerifier are required")
		return
	}

	tokenURL := s.cfg.KeycloakTokenURL
	if tokenURL == "" {
		tokenURL = authEndpoint(s.cfg.KeycloakIssuerURL, "token")
	}

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", s.cfg.KeycloakClientID)
	form.Set("code", req.Code)
	form.Set("redirect_uri", req.RedirectURI)
	form.Set("code_verifier", req.CodeVerifier)

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
	if !strings.HasPrefix(header, prefix) {
		return authUser{}, errors.New("Missing bearer token")
	}

	return s.validateAccessToken(strings.TrimSpace(strings.TrimPrefix(header, prefix)))
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
