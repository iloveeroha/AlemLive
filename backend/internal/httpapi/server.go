package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/iloveeroha/AlemLive/backend/internal/config"
	"github.com/iloveeroha/AlemLive/backend/internal/livekit"
)

type Server struct {
	cfg   config.Config
	clock func() time.Time
	mux   *http.ServeMux
}

type tokenRequest struct {
	RoomName string `json:"roomName"`
	Room     string `json:"room"`
	UserName string `json:"userName"`
	Identity string `json:"identity"`
}

type tokenResponse struct {
	ServerURL string `json:"serverUrl"`
	Token     string `json:"token"`
	RoomName  string `json:"roomName"`
	UserName  string `json:"userName"`
	ExpiresAt string `json:"expiresAt"`
}

func NewServer(cfg config.Config) http.Handler {
	server := &Server{
		cfg:   cfg,
		clock: time.Now,
		mux:   http.NewServeMux(),
	}

	server.routes()

	return server.withCORS(server.mux)
}

func (s *Server) routes() {
	s.mux.HandleFunc("/healthz", s.health)
	s.mux.HandleFunc("/api/config", s.config)
	s.mux.HandleFunc("/api/livekit/token", s.createLiveKitToken)
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) config(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"livekitUrl":    s.cfg.LiveKitURL,
		"tokenEndpoint": "/api/livekit/token",
	})
}

func (s *Server) createLiveKitToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}

	if s.cfg.LiveKitURL == "" || s.cfg.LiveKitAPIKey == "" || s.cfg.LiveKitSecret == "" {
		writeError(w, http.StatusServiceUnavailable, "LiveKit backend is not configured")
		return
	}

	var req tokenRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16*1024))
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	room := firstNonEmpty(req.RoomName, req.Room)
	identity := firstNonEmpty(req.UserName, req.Identity)

	room, err := validateField("roomName", room)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	identity, err = validateField("userName", identity)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	token, expiresAt, err := livekit.GenerateToken(
		s.cfg.LiveKitAPIKey,
		s.cfg.LiveKitSecret,
		identity,
		room,
		s.cfg.TokenTTL,
		s.clock(),
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not create LiveKit token")
		return
	}

	writeJSON(w, http.StatusOK, tokenResponse{
		ServerURL: s.cfg.LiveKitURL,
		Token:     token,
		RoomName:  room,
		UserName:  identity,
		ExpiresAt: expiresAt.UTC().Format(time.RFC3339),
	})
}

func (s *Server) withCORS(next http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(s.cfg.AllowedOrigins))
	allowAll := false

	for _, origin := range s.cfg.AllowedOrigins {
		if origin == "*" {
			allowAll = true
			continue
		}
		allowed[origin] = struct{}{}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			if allowAll {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			} else if _, ok := allowed[origin]; ok {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func validateField(name, value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New(name + " is required")
	}
	if len(value) > 128 {
		return "", errors.New(name + " is too long")
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return "", errors.New(name + " contains invalid characters")
		}
	}
	return value, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func methodNotAllowed(w http.ResponseWriter, allowed string) {
	w.Header().Set("Allow", allowed)
	writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
