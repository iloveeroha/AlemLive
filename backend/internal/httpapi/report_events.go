package httpapi

import (
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

type reportEventClient struct {
	conn *websocket.Conn
	send chan roomEventEnvelope
}

func (s *Server) reportEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}

	conn, err := roomEventUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := &reportEventClient{
		conn: conn,
		send: make(chan roomEventEnvelope, 16),
	}
	s.registerReportEventClient(client)
	defer s.unregisterReportEventClient(client)

	client.send <- roomEventEnvelope{
		Type: "connected",
		Payload: map[string]any{
			"status": "ok",
			"at":     s.clock().UTC().Format(time.RFC3339),
		},
	}

	go client.writePump()
	for {
		if _, _, err := conn.NextReader(); err != nil {
			return
		}
	}
}

func (s *Server) registerReportEventClient(client *reportEventClient) {
	s.reportClientsMu.Lock()
	defer s.reportClientsMu.Unlock()
	if s.reportClients == nil {
		s.reportClients = map[*reportEventClient]struct{}{}
	}
	s.reportClients[client] = struct{}{}
}

func (s *Server) unregisterReportEventClient(client *reportEventClient) {
	s.reportClientsMu.Lock()
	defer s.reportClientsMu.Unlock()
	if s.reportClients != nil {
		delete(s.reportClients, client)
	}
	close(client.send)
	_ = client.conn.Close()
}

func (s *Server) publishReportChanged(detail reportDetailResponse) {
	if detail.Report.ID == "" {
		return
	}

	payload := reportEventPayload(detail)
	s.broadcastReportEvent(roomEventEnvelope{Type: "report_changed", Payload: payload})

	if detail.RoomName != "" {
		s.broadcastRoomEvent(roomIDFromName(detail.RoomName), roomEventEnvelope{
			Type:    "report_status_changed",
			Payload: payload,
		})
	}
}

func (s *Server) publishReportDeleted(reportID string) {
	if reportID == "" {
		return
	}
	s.broadcastReportEvent(roomEventEnvelope{
		Type: "report_deleted",
		Payload: map[string]any{
			"reportId": reportID,
		},
	})
}

func reportEventPayload(detail reportDetailResponse) map[string]any {
	return map[string]any{
		"reportId":            detail.Report.ID,
		"roomName":            detail.RoomName,
		"report":              detail.Report,
		"status":              detail.Report.Status,
		"processingState":     detail.Report.ProcessingState,
		"recordingStatus":     detail.Report.RecordingStatus,
		"transcriptionStatus": detail.Report.TranscriptionStatus,
		"diarizationStatus":   detail.DiarizationStatus,
		"analysisStatus":      detail.Report.AnalysisStatus,
		"lastError":           detail.LastError,
	}
}

func (s *Server) broadcastReportEvent(event roomEventEnvelope) {
	s.reportClientsMu.Lock()
	defer s.reportClientsMu.Unlock()
	for client := range s.reportClients {
		select {
		case client.send <- event:
		default:
		}
	}
}

func (client *reportEventClient) writePump() {
	for event := range client.send {
		if err := client.conn.WriteJSON(event); err != nil {
			return
		}
	}
}
