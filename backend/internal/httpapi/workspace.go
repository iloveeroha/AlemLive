package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type meetingEventRequest struct {
	RoomName string `json:"roomName"`
	UserName string `json:"userName"`
	Event    string `json:"event"`
}

type devicePreferenceRequest struct {
	RoomName string `json:"roomName"`
	UserName string `json:"userName"`
	Device   string `json:"device"`
	Enabled  bool   `json:"enabled"`
}

func (s *Server) notifications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"unread": 1,
		"items": []map[string]string{
			{
				"id":        "livekit-ready",
				"title":     "LiveKit готов",
				"body":      "Backend выдаёт токены для комнат и готов к подключению фронта.",
				"createdAt": s.clock().UTC().Add(-15 * time.Minute).Format(time.RFC3339),
			},
		},
	})
}

func (s *Server) profile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}

	if user, ok := userFromContext(r.Context()); ok {
		initial := strings.ToUpper(firstNonEmpty(firstRune(user.Name), firstRune(user.Username), "U"))
		writeJSON(w, http.StatusOK, map[string]string{
			"id":       user.ID,
			"name":     user.Name,
			"email":    user.Email,
			"username": user.Username,
			"initial":  initial,
			"role":     "user",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"id":      "local-user",
		"name":    "Мади Орысбек",
		"initial": "М",
		"role":    "host",
	})
}

func (s *Server) locales(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"current": "ru",
		"items": []map[string]string{
			{"id": "ru", "label": "Русский"},
			{"id": "kk", "label": "Қазақша"},
			{"id": "en", "label": "English"},
		},
	})
}

func (s *Server) meetingEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}

	var req meetingEventRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16*1024))
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	roomName := strings.TrimSpace(firstNonEmpty(req.RoomName, "alem-meeting"))
	userName := strings.TrimSpace(firstNonEmpty(req.UserName, "Guest"))
	event := strings.TrimSpace(firstNonEmpty(req.Event, "unknown"))

	response := map[string]any{
		"roomName": roomName,
		"userName": userName,
		"event":    event,
		"status":   "recorded",
		"at":       s.clock().UTC().Format(time.RFC3339),
	}
	conference := s.recordConferenceEvent(roomName, userName, event, s.clock())
	if conference.ConferenceSaved {
		response["conference"] = conference
		response["reportId"] = conference.ReportID
	}
	switch strings.ToLower(event) {
	case "created", "joined":
		if state, err := s.startRoomRecording(r.Context(), roomName); err == nil {
			response["recording"] = state
			response["recordingStatus"] = "started"
		} else {
			response["recordingStatus"] = "not_started"
			response["recordingError"] = err.Error()
		}
	case "left", "ended":
		if conference.RecordingShouldStop {
			if state, err := s.stopRoomRecording(r.Context(), roomName); err == nil {
				response["recording"] = state
				response["recordingStatus"] = "stopped"
			} else {
				response["recordingStatus"] = "not_stopped"
				response["recordingError"] = err.Error()
			}
		} else {
			response["recordingStatus"] = "still_active"
		}
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) devicePreference(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}

	var req devicePreferenceRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16*1024))
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	device := strings.ToLower(strings.TrimSpace(req.Device))
	if device != "mic" && device != "camera" {
		writeError(w, http.StatusBadRequest, "device must be mic or camera")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"roomName": firstNonEmpty(req.RoomName, "alem-meeting"),
		"userName": firstNonEmpty(req.UserName, "Guest"),
		"device":   device,
		"enabled":  req.Enabled,
		"status":   "saved",
	})
}

func (s *Server) roomAction(w http.ResponseWriter, r *http.Request) {
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/rooms/"), "/")
	if rest == "" {
		writeError(w, http.StatusNotFound, "Room not found")
		return
	}

	parts := strings.Split(rest, "/")
	roomName, err := validateField("roomName", parts[0])
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	switch action {
	case "", "link":
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{
			"roomName": roomName,
			"url":      "/meeting?room=" + roomName,
			"joinUrl":  "/meeting?room=" + roomName,
		})
	case "settings":
		s.roomSettings(w, r, roomName)
	case "recording":
		subAction := ""
		if len(parts) > 2 {
			subAction = parts[2]
		}
		s.roomRecording(w, r, roomName, subAction)
	case "leave":
		if r.Method != http.MethodPost {
			methodNotAllowed(w, http.MethodPost)
			return
		}
		var req meetingEventRequest
		if r.Body != nil {
			_ = json.NewDecoder(http.MaxBytesReader(w, r.Body, 16*1024)).Decode(&req)
		}
		userName := strings.TrimSpace(firstNonEmpty(req.UserName, "Guest"))
		conference := s.recordConferenceEvent(roomName, userName, "left", s.clock())
		response := map[string]any{
			"roomName": roomName,
			"status":   "left",
			"at":       s.clock().UTC().Format(time.RFC3339),
		}
		if conference.ConferenceSaved {
			response["conference"] = conference
			response["reportId"] = conference.ReportID
		}
		if conference.RecordingShouldStop {
			if state, err := s.stopRoomRecording(r.Context(), roomName); err == nil {
				response["recording"] = state
				response["recordingStatus"] = "stopped"
			} else {
				response["recordingStatus"] = "not_stopped"
				response["recordingError"] = err.Error()
			}
		} else {
			response["recordingStatus"] = "still_active"
		}
		writeJSON(w, http.StatusOK, response)
	default:
		writeError(w, http.StatusNotFound, "Room action not found")
	}
}

func (s *Server) roomSettings(w http.ResponseWriter, r *http.Request, roomName string) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{
			"roomName":          roomName,
			"recording":         s.egress != nil && s.egress.Configured(),
			"transcription":     s.stt != nil && s.stt.Configured(),
			"waitingRoom":       false,
			"allowGuests":       true,
			"autoReport":        true,
			"reportEndpoint":    "/api/meetings/analysis?roomName=" + roomName,
			"livekitEndpoint":   "/api/livekit/token",
			"recordingEndpoint": "/api/rooms/" + roomName + "/recording",
		})
	case http.MethodPatch:
		var settings map[string]any
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16*1024))
		if err := decoder.Decode(&settings); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid JSON body")
			return
		}
		settings["roomName"] = roomName
		settings["status"] = "saved"
		writeJSON(w, http.StatusOK, settings)
	default:
		methodNotAllowed(w, http.MethodGet+", "+http.MethodPatch)
	}
}

func conferenceSummary(roomName, status string, egressEnabled bool) []summarySection {
	if status == "saved" {
		text := "Конференция сохранена в отчётах."
		if egressEnabled {
			text += " Запись, транскрипт и AI-анализ будут добавлены после завершения LiveKit Egress."
		} else {
			text += " Для видео и автоматического STT включите LiveKit Egress и storage."
		}
		return []summarySection{{Title: "Конференция сохранена", Text: text}}
	}

	text := "Backend уже создал отчёт для комнаты " + roomName + " и обновляет его по событиям участников."
	if egressEnabled {
		text += " LiveKit Egress записывает конференцию автоматически."
	}
	return []summarySection{{Title: "Конференция записывается", Text: text}}
}
