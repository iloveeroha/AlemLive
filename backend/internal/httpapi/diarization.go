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

	out := make([]transcriptLine, 0, len(lines))
	for _, line := range lines {
		out = append(out, splitTranscriptLineByDiarization(line, segments)...)
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

type diarizedLinePart struct {
	start   float64
	end     float64
	speaker string
}

func splitTranscriptLineByDiarization(line transcriptLine, segments []diarization.Segment) []transcriptLine {
	lineStart, lineEnd := transcriptBounds(line)
	parts := overlappingDiarizationParts(lineStart, lineEnd, segments)
	if len(parts) == 0 {
		line.Speaker = "SPEAKER_UNKNOWN"
		return []transcriptLine{line}
	}
	if len(parts) == 1 {
		line.Speaker = parts[0].speaker
		return []transcriptLine{line}
	}

	words := strings.Fields(line.Text)
	if len(words) < 2 {
		line.Speaker = bestDiarizationSpeaker(line, segments, "SPEAKER_UNKNOWN")
		return []transcriptLine{line}
	}

	chunks := splitWordsByDiarizationParts(words, parts)
	out := make([]transcriptLine, 0, len(chunks))
	for i, text := range chunks {
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		part := parts[i]
		next := line
		next.Time = formatTranscriptTime(part.start)
		next.Start = part.start
		next.End = part.end
		next.Speaker = part.speaker
		next.Text = text
		out = appendMergedTranscriptLine(out, next)
	}
	if len(out) == 0 {
		line.Speaker = bestDiarizationSpeaker(line, segments, "SPEAKER_UNKNOWN")
		return []transcriptLine{line}
	}
	return out
}

func overlappingDiarizationParts(lineStart, lineEnd float64, segments []diarization.Segment) []diarizedLinePart {
	parts := make([]diarizedLinePart, 0)
	for _, segment := range segments {
		start := maxFloat(lineStart, segment.Start)
		end := minFloat(lineEnd, segment.End)
		if end <= start {
			continue
		}
		parts = append(parts, diarizedLinePart{
			start:   start,
			end:     end,
			speaker: segment.Speaker,
		})
	}
	sort.SliceStable(parts, func(i, j int) bool {
		if parts[i].start == parts[j].start {
			return parts[i].end < parts[j].end
		}
		return parts[i].start < parts[j].start
	})
	return parts
}

func splitWordsByDiarizationParts(words []string, parts []diarizedLinePart) []string {
	chunks := make([]string, len(parts))
	totalWeight := 0.0
	for _, part := range parts {
		totalWeight += maxFloat(part.end-part.start, 0)
	}
	if totalWeight <= 0 {
		chunks[0] = strings.Join(words, " ")
		return chunks
	}

	wordIndex := 0
	totalWords := len(words)
	for i, part := range parts {
		remainingWords := totalWords - wordIndex
		remainingParts := len(parts) - i
		if remainingWords <= 0 {
			break
		}
		count := remainingWords
		if remainingParts > 1 {
			weight := maxFloat(part.end-part.start, 0)
			count = int(float64(totalWords)*weight/totalWeight + 0.5)
			if count < 1 {
				count = 1
			}
			maxAllowed := remainingWords - (remainingParts - 1)
			if maxAllowed < 1 {
				maxAllowed = 1
			}
			if count > maxAllowed {
				count = maxAllowed
			}
		}
		chunks[i] = strings.Join(words[wordIndex:wordIndex+count], " ")
		wordIndex += count
	}
	if wordIndex < totalWords {
		last := len(chunks) - 1
		chunks[last] = strings.TrimSpace(chunks[last] + " " + strings.Join(words[wordIndex:], " "))
	}
	return chunks
}

func appendMergedTranscriptLine(lines []transcriptLine, next transcriptLine) []transcriptLine {
	if len(lines) == 0 {
		return append(lines, next)
	}
	last := &lines[len(lines)-1]
	if normalizeSpeakerLabel(last.Speaker) != normalizeSpeakerLabel(next.Speaker) {
		return append(lines, next)
	}
	last.Text = strings.TrimSpace(last.Text + " " + next.Text)
	last.End = next.End
	return lines
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
	if len(names) > 0 {
		fillSpeakerNamesByParticipantOrder(speakerToName, usedNames, participantSpeakerOrder(lines, names), names)
	}
	fallbackNames := fallbackSpeakerNames(lines, speakerToName, names)
	for i, line := range lines {
		speaker := normalizeSpeakerLabel(line.Speaker)
		if known := matchKnownParticipantName(speaker, names); known != "" {
			line.Speaker = known
		} else if mapped, ok := speakerToName[speaker]; ok {
			line.Speaker = mapped
		} else if fallback, ok := fallbackNames[speaker]; ok {
			line.Speaker = fallback
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
		if _, ok := speakerToName[speaker]; ok {
			continue
		}
		if !isGenericSpeakerName(speaker) && (len(names) == 0 || matchKnownParticipantName(speaker, names) != "") {
			continue
		}
		name := selfIntroducedName(line.Text, names)
		if name == "" {
			continue
		}
		key := participantNameKey(name)
		if _, ok := usedNames[key]; ok {
			continue
		}
		speakerToName[speaker] = name
		usedNames[key] = struct{}{}
	}
	return speakerToName
}

func fillSpeakerNamesByParticipantOrder(speakerToName map[string]string, usedNames map[string]struct{}, speakers, names []string) {
	if len(speakers) == 0 || len(names) == 0 {
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
			key := participantNameKey(name)
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
		used[participantNameKey(name)] = struct{}{}
	}
	return used
}

func participantSpeakerOrder(lines []transcriptLine, names []string) []string {
	order := make([]string, 0)
	seen := map[string]struct{}{}
	for _, line := range lines {
		speaker := normalizeSpeakerLabel(line.Speaker)
		if matchKnownParticipantName(speaker, names) != "" {
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

func fallbackSpeakerNames(lines []transcriptLine, speakerToName map[string]string, names []string) map[string]string {
	fallback := map[string]string{}
	next := 1
	for _, line := range lines {
		speaker := normalizeSpeakerLabel(line.Speaker)
		if strings.EqualFold(speaker, "SPEAKER_UNKNOWN") {
			continue
		}
		if isGenericSpeakerName(speaker) || matchKnownParticipantName(speaker, names) != "" {
			continue
		}
		if _, ok := speakerToName[speaker]; ok {
			continue
		}
		if _, ok := fallback[speaker]; ok {
			continue
		}
		fallback[speaker] = "Speaker " + strconv.Itoa(next)
		next++
	}
	return fallback
}

func transcriptSpeakersNeedParticipantRepair(lines []transcriptLine, participants string) bool {
	names := participantNamesFromText(participants)
	if len(lines) == 0 {
		return false
	}
	if len(names) == 0 {
		for _, line := range lines {
			speaker := normalizeSpeakerLabel(line.Speaker)
			if !isGenericSpeakerName(speaker) && !strings.EqualFold(speaker, "SPEAKER_UNKNOWN") {
				return true
			}
		}
		return false
	}
	for _, line := range lines {
		speaker := normalizeSpeakerLabel(line.Speaker)
		if isGenericSpeakerName(speaker) || matchKnownParticipantName(speaker, names) == "" {
			return true
		}
	}
	return false
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
		key := participantNameKey(name)
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
	candidate = participantNameKey(candidate)
	if candidate == "" {
		return ""
	}
	candidateFirst := firstName(candidate)
	for _, name := range knownParticipants {
		name = strings.TrimSpace(name)
		lowerName := participantNameKey(name)
		if lowerName == candidate || strings.HasPrefix(lowerName, candidate+" ") || firstName(lowerName) == candidateFirst {
			return name
		}
	}
	return ""
}

func participantNameKey(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(value)), " "))
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
