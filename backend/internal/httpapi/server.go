package httpapi

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/iloveeroha/AlemLive/backend/internal/config"
	"github.com/iloveeroha/AlemLive/backend/internal/diarization"
	"github.com/iloveeroha/AlemLive/backend/internal/livekit"
	"github.com/iloveeroha/AlemLive/backend/internal/llm"
)

type Server struct {
	cfg      config.Config
	clock    func() time.Time
	ai       *llm.Client
	stt      *llm.Client
	diarizer *diarization.Client
	egress   *livekit.EgressManager
	mux      *http.ServeMux
	jwks     jwksCache

	reportsMu            sync.Mutex
	generatedReports     []reportRow
	generatedReportStore map[string]reportDetailResponse
	deletedReportIDs     map[string]struct{}
	activeMeetings       map[string]meetingSession
	latestRoomReports    map[string]string
	roomsMu              sync.Mutex
	rooms                map[string]*roomState
	roomClients          map[string]map[*roomEventClient]struct{}
	egressProcessingMu   sync.Mutex
	egressProcessing     map[string]struct{}
}

type tokenRequest struct {
	RoomName string `json:"roomName"`
	Room     string `json:"room"`
	UserName string `json:"userName"`
	Identity string `json:"identity"`
	IsHost   bool   `json:"isHost"`
}

type tokenResponse struct {
	ServerURL string `json:"serverUrl"`
	Token     string `json:"token"`
	RoomName  string `json:"roomName"`
	UserName  string `json:"userName"`
	ExpiresAt string `json:"expiresAt"`
	ReportID  string `json:"reportId,omitempty"`
}

type meetingAnalysis struct {
	RoomName     string           `json:"roomName"`
	GeneratedAt  string           `json:"generatedAt"`
	Summary      []summarySection `json:"summary"`
	ActionItems  []actionItem     `json:"actionItems"`
	KeyQuestions []keyQuestion    `json:"keyQuestions,omitempty"`
	Transcript   []transcriptLine `json:"transcript"`
	Insights     meetingInsights  `json:"insights"`
	Highlights   []highlight      `json:"highlights"`
	Chapters     []chapter        `json:"chapters"`
	Keywords     []string         `json:"keywords,omitempty"`
}

type summarySection struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

type actionItem struct {
	Time     string `json:"time,omitempty"`
	Task     string `json:"task"`
	Owner    string `json:"owner"`
	Due      string `json:"due"`
	Priority string `json:"priority"`
}

type keyQuestion struct {
	Time     string `json:"time,omitempty"`
	Question string `json:"question"`
	Answer   string `json:"answer,omitempty"`
}

type transcriptLine struct {
	Time      string  `json:"time"`
	Speaker   string  `json:"speaker"`
	Text      string  `json:"text"`
	Sentiment string  `json:"sentiment,omitempty"`
	Start     float64 `json:"-"`
	End       float64 `json:"-"`
}

type meetingInsights struct {
	Sentiment     string        `json:"sentiment"`
	TalkTime      []metricValue `json:"talkTime"`
	SpeechRate    []metricValue `json:"speechRate"`
	Interruptions []metricValue `json:"interruptions"`
	Engagement    []metricValue `json:"engagement"`
}

type metricValue struct {
	Label string `json:"label"`
	Value int    `json:"value"`
	Unit  string `json:"unit"`
}

type highlight struct {
	Time  string `json:"time"`
	Title string `json:"title"`
	Text  string `json:"text"`
	Note  string `json:"note,omitempty"`
	Type  string `json:"type"`
}

type chapter struct {
	Start    string   `json:"start"`
	End      string   `json:"end"`
	Time     string   `json:"time,omitempty"`
	Title    string   `json:"title"`
	Text     string   `json:"text"`
	Duration string   `json:"duration,omitempty"`
	Points   []string `json:"points,omitempty"`
}

func NewServer(cfg config.Config) http.Handler {
	sttBaseURL := firstNonEmpty(cfg.STTBaseURL, cfg.LLMBaseURL)
	sttAPIKey := firstNonEmpty(cfg.STTAPIKey, cfg.LLMAPIKey)
	sttModel := firstNonEmpty(cfg.STTModel, "whisper-1")
	sttTimeout := cfg.STTTimeout
	if sttTimeout <= 0 {
		sttTimeout = cfg.LLMTimeout
	}
	server := &Server{
		cfg:                  cfg,
		clock:                time.Now,
		ai:                   llm.New(cfg.LLMBaseURL, cfg.LLMAPIKey, cfg.LLMModel, cfg.LLMTimeout),
		stt:                  llm.New(sttBaseURL, sttAPIKey, sttModel, sttTimeout),
		diarizer:             diarization.New(cfg.DiarizationBaseURL, cfg.DiarizationAPIKey, cfg.DiarizationTimeout),
		egress:               livekit.NewEgressManager(egressConfigFromAppConfig(cfg)),
		mux:                  http.NewServeMux(),
		generatedReportStore: map[string]reportDetailResponse{},
		deletedReportIDs:     map[string]struct{}{},
		activeMeetings:       map[string]meetingSession{},
		latestRoomReports:    map[string]string{},
		rooms:                map[string]*roomState{},
		roomClients:          map[string]map[*roomEventClient]struct{}{},
		egressProcessing:     map[string]struct{}{},
	}
	server.cfg.STTBaseURL = sttBaseURL
	server.cfg.STTAPIKey = sttAPIKey
	server.cfg.STTModel = sttModel
	server.cfg.STTTimeout = sttTimeout
	server.loadReports()

	server.routes()

	return server.withCORS(server.withAuth(server.mux))
}

const askAIURL = "https://alem-workspace.gov.kz/web/alem-rag"

func (s *Server) routes() {
	s.mux.HandleFunc("/healthz", s.health)
	s.mux.HandleFunc("/api/config", s.config)
	s.mux.HandleFunc("/api/auth/config", s.authConfig)
	s.mux.HandleFunc("/api/auth/token", s.authToken)
	s.mux.HandleFunc("/api/auth/login", s.authLogin)
	s.mux.HandleFunc("/api/auth/register", s.authRegister)
	s.mux.HandleFunc("/api/auth/logout", s.authLogout)
	s.mux.HandleFunc("/api/auth/me", s.authMe)
	s.mux.HandleFunc("/api/livekit/webhook", s.liveKitWebhook)
	s.mux.HandleFunc("/api/livekit/token", s.createLiveKitToken)
	s.mux.HandleFunc("/api/meetings/analysis", s.meetingAnalysis)
	s.mux.HandleFunc("/api/meetings/transcribe", s.meetingTranscription)
	s.mux.HandleFunc("/api/meetings/events", s.meetingEvent)
	s.mux.HandleFunc("/api/rooms/create", s.createRoom)
	s.mux.HandleFunc("/api/rooms/join", s.joinRoom)
	s.mux.HandleFunc("/api/rooms/", s.roomAction)
	s.mux.HandleFunc("/api/devices", s.devicePreference)
	s.mux.HandleFunc("/api/notifications", s.notifications)
	s.mux.HandleFunc("/api/profile", s.profile)
	s.mux.HandleFunc("/api/locales", s.locales)
	s.mux.HandleFunc("/api/reports", s.reports)
	s.mux.HandleFunc("/api/reports/filters", s.reportFilters)
	s.mux.HandleFunc("/api/reports/upload", s.reportUpload)
	s.mux.HandleFunc("/api/reports/", s.reportByID)
	s.mux.HandleFunc("/api/ai/chat", s.aiChat)
	s.mux.HandleFunc("/api/ai/status", s.aiStatus)
	s.mux.HandleFunc("/api/ask-ai", s.askAI)
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

	writeJSON(w, http.StatusOK, map[string]any{
		"livekitUrl":             s.publicLiveKitURL(r),
		"tokenEndpoint":          "/api/livekit/token",
		"livekitWebhookEndpoint": "/api/livekit/webhook",
		"egressEnabled":          s.egress != nil && s.egress.Configured(),
		"aiChatEndpoint":         "/api/ai/chat",
		"aiStatusEndpoint":       "/api/ai/status",
		"analysisEndpoint":       "/api/meetings/analysis",
		"speechToTextEndpoint":   "/api/meetings/transcribe",
		"diarizationEnabled":     s.diarizer != nil && s.diarizer.Configured(),
		"meetingEventsEndpoint":  "/api/meetings/events",
		"roomsEndpoint":          "/api/rooms",
		"devicesEndpoint":        "/api/devices",
		"notificationsEndpoint":  "/api/notifications",
		"profileEndpoint":        "/api/profile",
		"localesEndpoint":        "/api/locales",
		"reportsEndpoint":        "/api/reports",
		"reportFiltersEndpoint":  "/api/reports/filters",
		"llmModel":               s.cfg.LLMModel,
		"sttModel":               s.cfg.STTModel,
	})
}

func egressConfigFromAppConfig(cfg config.Config) livekit.EgressConfig {
	return livekit.EgressConfig{
		Enabled:       cfg.LiveKitEgressEnabled,
		ServerURL:     cfg.LiveKitURL,
		APIKey:        cfg.LiveKitAPIKey,
		APISecret:     cfg.LiveKitSecret,
		AudioOnly:     cfg.LiveKitEgressAudioOnly,
		Layout:        cfg.LiveKitEgressLayout,
		FilePrefix:    cfg.LiveKitEgressFilePrefix,
		PublicBaseURL: cfg.LiveKitEgressPublicBaseURL,
		WebhookURL:    cfg.LiveKitEgressWebhookURL,
		S3: livekit.S3Config{
			AccessKey:      cfg.LiveKitS3AccessKey,
			Secret:         cfg.LiveKitS3Secret,
			SessionToken:   cfg.LiveKitS3SessionToken,
			Region:         cfg.LiveKitS3Region,
			Endpoint:       cfg.LiveKitS3Endpoint,
			Bucket:         cfg.LiveKitS3Bucket,
			ForcePathStyle: cfg.LiveKitS3ForcePathStyle,
		},
	}
}

func (s *Server) authConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}

	issuer := s.cfg.KeycloakIssuerURL
	writeJSON(w, http.StatusOK, map[string]any{
		"enabled":               s.authEnabled(),
		"issuerUrl":             issuer,
		"clientId":              s.cfg.KeycloakClientID,
		"authorizationEndpoint": authEndpoint(issuer, "auth"),
		"tokenEndpoint":         "/api/auth/token",
		"logoutEndpoint":        authEndpoint(issuer, "logout"),
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

	user := roomUser{ID: identity, Name: identity}
	snapshot, _ := s.joinRoomState(room, user, req.IsHost, true, true)
	conferenceEvent := "joined"
	if req.IsHost {
		conferenceEvent = "created"
	}
	conference := s.recordConferenceEvent(snapshot.Name, user.Name, conferenceEvent, s.clock())
	if conference.ReportID != "" {
		snapshot = s.setRoomRecordingState(snapshot.ID, snapshot.RecordingState, conference.ReportID)
	}

	role := "participant"
	if req.IsHost {
		role = "host"
	}
	metadata, err := json.Marshal(map[string]string{"role": role})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not create LiveKit token")
		return
	}

	token, expiresAt, err := livekit.GenerateToken(
		s.cfg.LiveKitAPIKey,
		s.cfg.LiveKitSecret,
		identity,
		room,
		string(metadata),
		s.cfg.TokenTTL,
		s.clock(),
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not create LiveKit token")
		return
	}

	writeJSON(w, http.StatusOK, tokenResponse{
		ServerURL: s.publicLiveKitURL(r),
		Token:     token,
		RoomName:  room,
		UserName:  identity,
		ExpiresAt: expiresAt.UTC().Format(time.RFC3339),
		ReportID:  conference.ReportID,
	})
}

func (s *Server) publicLiveKitURL(r *http.Request) string {
	explicitURL := strings.TrimSpace(s.cfg.LiveKitPublicURL)
	host := forwardedHost(r)
	if explicitURL != "" {
		if isLoopbackURL(explicitURL) && host != "" && !isLoopbackHost(stripPort(host)) {
			return liveKitURLWithRequestHost(explicitURL, host)
		}
		return explicitURL
	}

	if host == "" {
		return s.cfg.LiveKitURL
	}

	scheme := "ws"
	if requestScheme(r) == "https" {
		scheme = "wss"
	}

	return scheme + "://" + host + "/livekit"
}

func liveKitURLWithRequestHost(rawURL, requestHost string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme == "" {
		return rawURL
	}
	host := stripPort(requestHost)
	if host == "" {
		return rawURL
	}
	port := parsed.Port()
	if port != "" {
		parsed.Host = net.JoinHostPort(host, port)
	} else {
		parsed.Host = host
	}
	return parsed.String()
}

func stripPort(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}
	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		return parsedHost
	}
	return strings.Trim(host, "[]")
}

func forwardedHost(r *http.Request) string {
	if r == nil {
		return ""
	}
	forwardedHost := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Host"), ",")[0])
	if forwardedHost != "" {
		return forwardedHost
	}
	return strings.TrimSpace(r.Host)
}

func requestScheme(r *http.Request) string {
	if r == nil {
		return "http"
	}
	if proto := strings.ToLower(strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Proto"), ",")[0])); proto != "" {
		return proto
	}
	if r.TLS != nil {
		return "https"
	}
	return "http"
}

func isLoopbackURL(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return isLoopbackHost(parsed.Hostname())
}

func isLoopbackHost(host string) bool {
	host = strings.Trim(strings.ToLower(strings.TrimSpace(host)), "[]")
	if host == "localhost" || host == "::1" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (s *Server) askAI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"url": askAIURL})
}

func (s *Server) meetingAnalysis(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}

	room := strings.TrimSpace(r.URL.Query().Get("roomName"))
	if room == "" {
		room = "alem-meeting"
	}

	room, err := validateField("roomName", room)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if s.ai != nil && s.ai.Configured() {
		if analysis, err := s.generateMeetingAnalysis(r.Context(), room, s.clock); err == nil {
			writeJSON(w, http.StatusOK, analysis)
			return
		}
	}

	writeJSON(w, http.StatusOK, demoMeetingAnalysis(room, s.clock()))
}

func demoMeetingAnalysis(room string, now time.Time) meetingAnalysis {
	return meetingAnalysis{
		RoomName:    room,
		GeneratedAt: now.UTC().Format(time.RFC3339),
		Summary: []summarySection{
			{
				Title: "Product scope",
				Text:  "Команда согласовала базовый сценарий: LiveKit комната, backend-issued токены и post-meeting AI workspace для заметок, транскрипта и аналитики.",
			},
			{
				Title: "Backend and deploy",
				Text:  "Go backend отвечает за безопасную выдачу LiveKit токенов, Docker Compose поднимает frontend и backend одной командой.",
			},
			{
				Title: "Next AI layer",
				Text:  "После записи встречи pipeline должен превратить аудио в transcript, затем LLM сформирует summary, chapters, highlights и action items.",
			},
		},
		ActionItems: []actionItem{
			{Task: "Подключить запись и хранение meeting artifacts", Owner: "Backend", Due: "Friday", Priority: "High"},
			{Task: "Согласовать формат transcript JSON", Owner: "AI team", Due: "Monday", Priority: "High"},
			{Task: "Добавить экспорт summary в PDF/DOCX", Owner: "Frontend", Due: "Next sprint", Priority: "Medium"},
		},
		Transcript: []transcriptLine{
			{Time: "00:00", Speaker: "Host", Text: "Запускаем встречу и проверяем LiveKit подключение."},
			{Time: "03:20", Speaker: "Frontend", Text: "Нужны отдельные вкладки Notes, Transcript, Insights, Highlights и Chapters."},
			{Time: "08:45", Speaker: "Backend", Text: "Токены должны создаваться только на сервере, секреты не уходят в браузер."},
			{Time: "14:10", Speaker: "AI", Text: "После звонка транскрипт можно отправить в LLM для краткого саммари и задач."},
			{Time: "21:35", Speaker: "Team", Text: "Docker Compose должен запускать весь стек одной командой."},
		},
		Insights: meetingInsights{
			Sentiment: "Constructive",
			TalkTime: []metricValue{
				{Label: "Host", Value: 38, Unit: "%"},
				{Label: "Frontend", Value: 27, Unit: "%"},
				{Label: "Backend", Value: 24, Unit: "%"},
				{Label: "AI", Value: 11, Unit: "%"},
			},
			SpeechRate: []metricValue{
				{Label: "Host", Value: 142, Unit: "wpm"},
				{Label: "Frontend", Value: 128, Unit: "wpm"},
				{Label: "Backend", Value: 118, Unit: "wpm"},
			},
			Interruptions: []metricValue{
				{Label: "Host", Value: 1, Unit: "times"},
				{Label: "Frontend", Value: 0, Unit: "times"},
				{Label: "Backend", Value: 1, Unit: "times"},
			},
			Engagement: []metricValue{
				{Label: "Questions", Value: 8, Unit: "items"},
				{Label: "Decisions", Value: 5, Unit: "items"},
				{Label: "Action items", Value: 3, Unit: "items"},
			},
		},
		Highlights: []highlight{
			{Time: "03:20", Title: "Post-meeting sections defined", Text: "Команда перечислила пять основных вкладок продукта.", Type: "Decision"},
			{Time: "08:45", Title: "Security boundary", Text: "LiveKit API secret остается только на backend.", Type: "Risk"},
			{Time: "21:35", Title: "Docker launch agreed", Text: "Стек должен стартовать через docker compose up -d --build.", Type: "Action"},
		},
		Chapters: []chapter{
			{
				Start: "00:00", End: "03:19", Title: "Setup and room check",
				Text:   "Проверка LiveKit комнаты и подключения участников перед началом основной части встречи.",
				Points: []string{"Подключение участников к LiveKit комнате", "Проверка аудио и видео перед началом"},
			},
			{
				Start: "03:20", End: "08:44", Title: "Product requirements",
				Text:   "Команда разобрала вкладки отчёта о встрече: Notes, Transcript, Insights, Highlights и Chapters.",
				Points: []string{"Структура вкладок post-meeting отчёта", "Что показывается в каждой вкладке"},
			},
			{
				Start: "08:45", End: "14:09", Title: "Backend security",
				Text:   "Обсуждение токенов, секретов и границы между фронтендом и backend.",
				Points: []string{"LiveKit токены выдаются только сервером", "Секреты не попадают в браузер"},
			},
			{
				Start: "14:10", End: "21:34", Title: "AI processing pipeline",
				Text:   "Как транскрипт превращается в summary, highlights и action items с помощью LLM.",
				Points: []string{"STT переводит запись в текст", "LLM формирует сводку, главы и задачи"},
			},
			{
				Start: "21:35", End: "25:00", Title: "Deployment",
				Text:   "Docker Compose поднимает весь стек и production frontend через nginx.",
				Points: []string{"docker compose up -d --build запускает весь стек", "Frontend раздаётся через nginx"},
			},
		},
		Keywords: []string{"livekit", "комната", "backend", "docker", "транскрипт", "токены"},
	}
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
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
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

func authEndpoint(issuerURL, action string) string {
	if issuerURL == "" {
		return ""
	}
	return strings.TrimRight(issuerURL, "/") + "/protocol/openid-connect/" + action
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
