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
	writeJSON(w, http.StatusOK, map[string]string{
		"roomName": roomName,
		"userName": userName,
		"event":    event,
		"status":   "recorded",
		"at":       s.clock().UTC().Format(time.RFC3339),
	})
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
	case "leave":
		if r.Method != http.MethodPost {
			methodNotAllowed(w, http.MethodPost)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{
			"roomName": roomName,
			"status":   "left",
			"at":       s.clock().UTC().Format(time.RFC3339),
		})
	default:
		writeError(w, http.StatusNotFound, "Room action not found")
	}
}

func (s *Server) roomSettings(w http.ResponseWriter, r *http.Request, roomName string) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{
			"roomName":        roomName,
			"recording":       true,
			"transcription":   true,
			"waitingRoom":     false,
			"allowGuests":     true,
			"autoReport":      true,
			"reportEndpoint":  "/api/meetings/analysis?roomName=" + roomName,
			"livekitEndpoint": "/api/livekit/token",
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
