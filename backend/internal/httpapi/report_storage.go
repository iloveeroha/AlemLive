package httpapi

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type persistedReports struct {
	Reports          []reportDetailResponse `json:"reports"`
	DeletedReportIDs []string               `json:"deletedReportIds,omitempty"`
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

	changed := false
	if s.deletedReportIDs == nil {
		s.deletedReportIDs = map[string]struct{}{}
	}
	for _, id := range payload.DeletedReportIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			s.deletedReportIDs[id] = struct{}{}
		}
	}
	for _, detail := range payload.Reports {
		if detail.Report.ID == "" {
			continue
		}
		if _, deleted := s.deletedReportIDs[detail.Report.ID]; deleted {
			continue
		}
		if normalizeLoadedReport(&detail) {
			changed = true
		}
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
	if changed {
		s.saveReportsLocked()
	}
}

func normalizeLoadedReport(detail *reportDetailResponse) bool {
	changed := false
	if detail.Report.OccurredAt.IsZero() && detail.Report.CreatedAt != "" {
		if parsed, err := time.Parse(time.RFC3339, detail.Report.CreatedAt); err == nil {
			detail.Report.OccurredAt = parsed
			changed = true
		}
	}
	if detail.Transcript == nil {
		detail.Transcript = detail.TranscriptLines
		changed = true
	}
	if detail.RecordingFile != "" && detail.RecordingURL == "" && detail.Report.ID != "" {
		detail.RecordingURL = "/api/reports/" + detail.Report.ID + "/recording/stream"
		changed = true
	}
	if repairLoadedReportAnalysis(detail) {
		changed = true
	}
	return changed
}

func repairLoadedReportAnalysis(detail *reportDetailResponse) bool {
	lines := reportTranscriptToLines(firstNonEmptyTranscript(detail.TranscriptLines, detail.Transcript))
	if len(lines) == 0 {
		return false
	}

	needsSpeakerRepair := hasOnlyGenericReportSpeakers(detail.TranscriptLines) && hasOnlyGenericReportSpeakers(detail.Transcript)
	needsFallbackRepair := isFallbackReportAnalysis(*detail)
	if !needsSpeakerRepair && !needsFallbackRepair {
		return false
	}

	lines = normalizeTranscriptSpeakers(lines)
	transcriptText := transcriptTextFromLines(lines)

	if needsFallbackRepair {
		analysis := fallbackAnalysisFromTranscript(firstNonEmpty(detail.RoomName, detail.Report.Title, detail.Report.ID), transcriptText, lines, detail.Report.OccurredAt)
		updated := reportDetailFromAnalysis(detail.Report, analysis)
		updated.RecordingURL = detail.RecordingURL
		updated.RecordingFile = detail.RecordingFile
		updated.RecordingType = detail.RecordingType
		updated.RoomName = firstNonEmpty(updated.RoomName, detail.RoomName)
		*detail = updated
		return true
	}

	repairedTranscript := reportLinesFromTranscript(lines)
	detail.TranscriptLines = repairedTranscript
	detail.Transcript = repairedTranscript
	if len(detail.SpeakerStats) == 0 || hasOnlyGenericSpeakerStats(reportSpeakerStatsToMetrics(detail.SpeakerStats)) {
		detail.SpeakerStats = speakerStatsFromMetrics(speakerTalkTime(lines))
	}
	return true
}

func firstNonEmptyTranscript(values ...[]reportTranscript) []reportTranscript {
	for _, value := range values {
		if len(value) > 0 {
			return value
		}
	}
	return nil
}

func reportTranscriptToLines(items []reportTranscript) []transcriptLine {
	lines := make([]transcriptLine, 0, len(items))
	for _, item := range items {
		lines = append(lines, transcriptLine{Time: item.Time, Speaker: item.Speaker, Text: item.Text})
	}
	return lines
}

func reportLinesFromTranscript(lines []transcriptLine) []reportTranscript {
	items := make([]reportTranscript, 0, len(lines))
	for i, line := range lines {
		items = append(items, reportTranscript{
			ID:      fmt.Sprintf("t%d", i+1),
			Time:    line.Time,
			Speaker: line.Speaker,
			Text:    line.Text,
		})
	}
	return items
}

func transcriptTextFromLines(lines []transcriptLine) string {
	parts := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line.Text) != "" {
			parts = append(parts, strings.TrimSpace(line.Text))
		}
	}
	return strings.Join(parts, ". ")
}

func hasOnlyGenericReportSpeakers(items []reportTranscript) bool {
	if len(items) == 0 {
		return true
	}
	for _, item := range items {
		if !isGenericSpeakerName(item.Speaker) {
			return false
		}
	}
	return true
}

func isFallbackReportAnalysis(detail reportDetailResponse) bool {
	if len(detail.Summary) == 1 && strings.EqualFold(strings.TrimSpace(detail.Summary[0].Title), "Transcript captured") {
		return true
	}
	if len(detail.ActionItems) == 1 && strings.Contains(strings.ToLower(detail.ActionItems[0].Title+" "+detail.ActionItems[0].Task), "review generated transcript") {
		return true
	}
	return false
}

func reportSpeakerStatsToMetrics(stats []speakerStat) []metricValue {
	values := make([]metricValue, 0, len(stats))
	for _, stat := range stats {
		values = append(values, metricValue{Label: stat.Name, Value: stat.TalkTime, Unit: "%"})
	}
	return values
}

func speakerStatsFromMetrics(metrics []metricValue) []speakerStat {
	stats := make([]speakerStat, 0, len(metrics))
	for _, metric := range metrics {
		stats = append(stats, speakerStat{
			Name:         metric.Label,
			Role:         "Speaker",
			TalkTime:     metric.Value,
			TalkTimeText: fmt.Sprintf("%d%s", metric.Value, metric.Unit),
			Talk:         metric.Value,
			Sentiment:    "Информационная встреча",
		})
	}
	return stats
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
	deletedIDs := make([]string, 0, len(s.deletedReportIDs))
	for id := range s.deletedReportIDs {
		deletedIDs = append(deletedIDs, id)
	}
	sort.Strings(deletedIDs)

	data, err := json.MarshalIndent(persistedReports{
		Reports:          reports,
		DeletedReportIDs: deletedIDs,
	}, "", "  ")
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
