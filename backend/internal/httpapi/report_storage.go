package httpapi

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type persistedReports struct {
	Reports []reportDetailResponse `json:"reports"`
}

func (s *Server) loadReports() {
	if s.cfg.ReportsStoragePath == "" {
		return
	}

	data, err := os.ReadFile(s.cfg.ReportsStoragePath)
	if err != nil {
		return
	}

	var payload persistedReports
	if err := json.Unmarshal(data, &payload); err != nil {
		return
	}

	s.reportsMu.Lock()
	defer s.reportsMu.Unlock()

	for _, detail := range payload.Reports {
		if detail.Report.ID == "" {
			continue
		}
		normalizeLoadedReport(&detail)
		s.generatedReports = append(s.generatedReports, detail.Report)
		s.generatedReportStore[detail.Report.ID] = detail
	}

	sort.SliceStable(s.generatedReports, func(i, j int) bool {
		return s.generatedReports[i].OccurredAt.After(s.generatedReports[j].OccurredAt)
	})
	for _, row := range s.generatedReports {
		detail := s.generatedReportStore[row.ID]
		if detail.RoomName != "" && s.latestRoomReports[detail.RoomName] == "" {
			s.latestRoomReports[detail.RoomName] = detail.Report.ID
		}
	}
}

func normalizeLoadedReport(detail *reportDetailResponse) {
	if detail.Report.OccurredAt.IsZero() && detail.Report.CreatedAt != "" {
		if parsed, err := time.Parse(time.RFC3339, detail.Report.CreatedAt); err == nil {
			detail.Report.OccurredAt = parsed
		}
	}
	if detail.Transcript == nil {
		detail.Transcript = detail.TranscriptLines
	}
	if detail.RecordingFile != "" && detail.RecordingURL == "" && detail.Report.ID != "" {
		detail.RecordingURL = "/api/reports/" + detail.Report.ID + "/recording/stream"
	}
}

func (s *Server) saveReportsLocked() {
	if s.cfg.ReportsStoragePath == "" {
		return
	}

	reports := make([]reportDetailResponse, 0, len(s.generatedReports))
	for _, row := range s.generatedReports {
		if detail, ok := s.generatedReportStore[row.ID]; ok {
			reports = append(reports, detail)
		}
	}

	data, err := json.MarshalIndent(persistedReports{Reports: reports}, "", "  ")
	if err != nil {
		return
	}

	dir := filepath.Dir(s.cfg.ReportsStoragePath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return
		}
	}

	tmp := s.cfg.ReportsStoragePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return
	}
	_ = os.Rename(tmp, s.cfg.ReportsStoragePath)
}
