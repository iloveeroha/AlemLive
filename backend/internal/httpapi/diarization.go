package httpapi

import (
	"context"
	"log"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/iloveeroha/AlemLive/backend/internal/diarization"
)

var diarizationSpeakerPattern = regexp.MustCompile(`(?i)^(?:speaker|spk)[_\s-]*(\d+)$`)
var selfIntroductionPattern = regexp.MustCompile(`(?i)(?:\b(?:i['’]?m|i\s+am|my\s+name\s+is|this\s+is)|(?:^|[^\p{L}])(?:я|меня\s+зовут|это))\s+([\p{L}][\p{L}'’.-]*(?:\s+[\p{L}][\p{L}'’.-]*){0,2})`)

func (s *Server) diarizeTranscript(ctx context.Context, fileName, contentType string, data []byte, participants string, lines []transcriptLine) []transcriptLine {
	if s.diarizer == nil || !s.diarizer.Configured() || len(data) == 0 || len(lines) == 0 {
		return lines
	}

	result, err := s.diarizer.Diarize(ctx, fileName, contentType, data, diarization.Options{
		Participants: participants,
	})
	if err != nil {
		log.Printf("diarization skipped: %v", err)
		return lines
	}

	return applyDiarizationSegments(lines, result.Segments, participants)
}

func applyDiarizationSegments(lines []transcriptLine, segments []diarization.Segment, participants ...string) []transcriptLine {
	if len(lines) == 0 || len(segments) == 0 {
		return lines
	}

	segments = normalizedDiarizationSegments(segments)
	if len(segments) == 0 {
		return lines
	}

	out := make([]transcriptLine, len(lines))
	for i, line := range lines {
		line.Speaker = bestDiarizationSpeaker(line, segments, line.Speaker)
		out[i] = line
	}
	out = normalizeTranscriptSpeakers(out)
	out = applyParticipantSpeakerNames(out, strings.Join(participants, ", "))
	return out
}

func normalizedDiarizationSegments(segments []diarization.Segment) []diarization.Segment {
	out := make([]diarization.Segment, 0, len(segments))
	for _, segment := range segments {
		segment.Speaker = normalizeDiarizationSpeaker(segment.Speaker)
		if segment.Speaker == "" || segment.End <= segment.Start {
			continue
		}
		out = append(out, segment)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Start < out[j].Start
	})
	return out
}

func bestDiarizationSpeaker(line transcriptLine, segments []diarization.Segment, fallback string) string {
	lineStart, lineEnd := transcriptBounds(line)
	bestSpeaker := ""
	bestOverlap := 0.0

	for _, segment := range segments {
		overlap := overlapSeconds(lineStart, lineEnd, segment.Start, segment.End)
		if overlap > bestOverlap {
			bestOverlap = overlap
			bestSpeaker = segment.Speaker
		}
	}
	if bestSpeaker != "" {
		return bestSpeaker
	}

	midpoint := lineStart + (lineEnd-lineStart)/2
	for _, segment := range segments {
		if midpoint >= segment.Start && midpoint <= segment.End {
			return segment.Speaker
		}
	}

	return fallback
}

func transcriptBounds(line transcriptLine) (float64, float64) {
	start := line.Start
	end := line.End
	if end <= start {
		end = start + 0.1
	}
	if start < 0 {
		start = 0
	}
	if end < start {
		end = start
	}
	return start, end
}

func overlapSeconds(aStart, aEnd, bStart, bEnd float64) float64 {
	start := maxFloat(aStart, bStart)
	end := minFloat(aEnd, bEnd)
	if end <= start {
		return 0
	}
	return end - start
}

func normalizeDiarizationSpeaker(speaker string) string {
	speaker = strings.TrimSpace(speaker)
	if speaker == "" {
		return ""
	}
	speaker = strings.ReplaceAll(speaker, "_", " ")
	speaker = strings.Join(strings.Fields(speaker), " ")

	if matches := diarizationSpeakerPattern.FindStringSubmatch(speaker); len(matches) == 2 {
		n, err := strconv.Atoi(matches[1])
		if err == nil {
			if n == 0 || strings.HasPrefix(matches[1], "0") {
				n++
			}
			return "Speaker " + strconv.Itoa(n)
		}
	}

	if strings.EqualFold(speaker, "speaker") {
		return "Speaker"
	}
	return speaker
}

func applyParticipantSpeakerNames(lines []transcriptLine, participants string) []transcriptLine {
	names := participantNamesFromText(participants)
	if len(lines) == 0 {
		return lines
	}

	out := make([]transcriptLine, len(lines))
	speakerToName := inferSpeakerNamesFromTranscript(lines, names)
	usedNames := usedSpeakerNames(speakerToName)
	if len(names) > 1 {
		fillSpeakerNamesByParticipantOrder(speakerToName, usedNames, genericSpeakerOrder(lines), names)
	}
	for i, line := range lines {
		speaker := normalizeSpeakerLabel(line.Speaker)
		if isGenericSpeakerName(speaker) {
			if mapped, ok := speakerToName[speaker]; ok {
				line.Speaker = mapped
			} else {
				line.Speaker = speaker
			}
		} else {
			line.Speaker = speaker
		}
		out[i] = line
	}
	return out
}

func inferSpeakerNamesFromTranscript(lines []transcriptLine, names []string) map[string]string {
	speakerToName := map[string]string{}
	usedNames := map[string]struct{}{}
	for _, line := range lines {
		speaker := normalizeSpeakerLabel(line.Speaker)
		if !isGenericSpeakerName(speaker) {
			continue
		}
		if _, ok := speakerToName[speaker]; ok {
			continue
		}
		name := selfIntroducedName(line.Text, names)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, ok := usedNames[key]; ok {
			continue
		}
		speakerToName[speaker] = name
		usedNames[key] = struct{}{}
	}
	return speakerToName
}

func fillSpeakerNamesByParticipantOrder(speakerToName map[string]string, usedNames map[string]struct{}, speakers, names []string) {
	if len(speakers) == 0 || len(names) < len(speakers) {
		return
	}
	nextName := 0
	for _, speaker := range speakers {
		if _, ok := speakerToName[speaker]; ok {
			continue
		}
		for nextName < len(names) {
			name := names[nextName]
			nextName++
			key := strings.ToLower(name)
			if _, used := usedNames[key]; used {
				continue
			}
			speakerToName[speaker] = name
			usedNames[key] = struct{}{}
			break
		}
	}
}

func usedSpeakerNames(speakerToName map[string]string) map[string]struct{} {
	used := map[string]struct{}{}
	for _, name := range speakerToName {
		used[strings.ToLower(name)] = struct{}{}
	}
	return used
}

func genericSpeakerOrder(lines []transcriptLine) []string {
	order := make([]string, 0)
	seen := map[string]struct{}{}
	for _, line := range lines {
		speaker := normalizeSpeakerLabel(line.Speaker)
		if !isGenericSpeakerName(speaker) {
			continue
		}
		if _, ok := seen[speaker]; ok {
			continue
		}
		seen[speaker] = struct{}{}
		order = append(order, speaker)
	}
	return order
}

func participantNamesFromText(value string) []string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == '\r' || r == '\t'
	})

	names := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		name := strings.Trim(part, " \t\r\n()[]{}")
		name = strings.Join(strings.Fields(name), " ")
		if !looksLikeParticipantName(name) {
			continue
		}
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		names = append(names, name)
	}
	return names
}

func looksLikeParticipantName(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" || strings.HasPrefix(name, "+") {
		return false
	}
	lower := strings.ToLower(name)
	blocked := []string{
		"more",
		"others",
		"больше",
		"друг",
		"ожидает анализа",
		"livekit room",
		"processed",
	}
	for _, word := range blocked {
		if strings.Contains(lower, word) {
			return false
		}
	}
	_, err := strconv.Atoi(name)
	return err != nil
}

func selfIntroducedName(text string, knownParticipants []string) string {
	for _, match := range selfIntroductionPattern.FindAllStringSubmatch(text, -1) {
		if len(match) < 2 {
			continue
		}
		candidate := cleanIntroducedName(match[1])
		if candidate == "" {
			continue
		}
		if known := matchKnownParticipantName(candidate, knownParticipants); known != "" {
			return known
		}
		return titleName(candidate)
	}
	return ""
}

func cleanIntroducedName(value string) string {
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return ""
	}

	parts := make([]string, 0, 2)
	for _, field := range fields {
		field = strings.Trim(field, " \t\r\n,.:;!?()[]{}\"'`")
		field = strings.Trim(field, "’")
		if field == "" {
			continue
		}
		lower := strings.ToLower(field)
		if isNameStopWord(lower) {
			break
		}
		parts = append(parts, field)
		if len(parts) >= 2 {
			break
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " ")
}

func isNameStopWord(value string) bool {
	switch value {
	case "a", "an", "am", "and", "at", "for", "from", "going", "gonna", "happy", "here", "i", "in", "me", "my", "on", "the", "to", "we", "with",
		"буду", "в", "и", "из", "меня", "на", "с", "это", "я":
		return true
	default:
		return false
	}
}

func matchKnownParticipantName(candidate string, knownParticipants []string) string {
	candidate = strings.ToLower(strings.TrimSpace(candidate))
	if candidate == "" {
		return ""
	}
	candidateFirst := firstName(candidate)
	for _, name := range knownParticipants {
		name = strings.TrimSpace(name)
		lowerName := strings.ToLower(name)
		if lowerName == candidate || strings.HasPrefix(lowerName, candidate+" ") || firstName(lowerName) == candidateFirst {
			return name
		}
	}
	return ""
}

func firstName(value string) string {
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return ""
	}
	return strings.Trim(fields[0], " \t\r\n,.:;!?()[]{}\"'`")
}

func titleName(value string) string {
	fields := strings.Fields(value)
	for i, field := range fields {
		runes := []rune(strings.ToLower(field))
		if len(runes) == 0 {
			continue
		}
		runes[0] = unicode.ToUpper(runes[0])
		fields[i] = string(runes)
	}
	return strings.Join(fields, " ")
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
