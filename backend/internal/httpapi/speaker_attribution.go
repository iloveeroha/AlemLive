package httpapi

import (
	"fmt"
	"sort"
	"strings"
)

const (
	speakerSourceSTT              = "stt"
	speakerSourceDiarization      = "diarization"
	speakerSourceParticipantTrack = "participant_track"
	speakerSourceManual           = "manual"
	speakerSourceUnknown          = "unknown"
)

func (s *Server) syncReportParticipantsFromSnapshot(reportID string, snapshot roomStateSnapshot, extra ...*roomParticipantState) {
	reportID = strings.TrimSpace(reportID)
	if reportID == "" {
		return
	}
	incoming := reportParticipantsFromSnapshot(snapshot, extra...)
	if len(incoming) == 0 {
		return
	}

	detail, ok := s.reportDetailForUpdate(reportID)
	if !ok {
		return
	}
	detail.Participants = mergeMeetingParticipants(detail.Participants, incoming)
	detail.Report.Participants = len(detail.Participants)
	if names := participantDisplayNames(detail.Participants); names != "" {
		detail.Report.ParticipantNames = names
	}
	detail.RoomName = firstNonEmpty(detail.RoomName, snapshot.Name)
	s.storeReport(detail)
}

func reportParticipantsFromSnapshot(snapshot roomStateSnapshot, extra ...*roomParticipantState) []meetingParticipant {
	items := make([]meetingParticipant, 0, len(snapshot.Participants)+len(extra))
	for _, participant := range snapshot.Participants {
		items = append(items, reportParticipantFromRoomParticipant(participant))
	}
	for _, participant := range extra {
		items = append(items, reportParticipantFromRoomParticipant(participant))
	}
	out := items[:0]
	for _, item := range items {
		if item.ParticipantID == "" && item.DisplayName == "" && item.LiveKitIdentity == "" {
			continue
		}
		out = append(out, item)
	}
	return out
}

func reportParticipantFromRoomParticipant(participant *roomParticipantState) meetingParticipant {
	if participant == nil {
		return meetingParticipant{}
	}
	return meetingParticipant{
		ParticipantID:   participant.ID,
		LiveKitIdentity: participant.ID,
		DisplayName:     participant.Name,
		Email:           participant.Email,
		JoinedAt:        formatOptionalTime(participant.JoinedAt),
		LeftAt:          formatOptionalTime(participant.LeftAt),
		AudioTrackIDs:   append([]string(nil), participant.AudioTrackIDs...),
	}
}

func mergeMeetingParticipants(existing, incoming []meetingParticipant) []meetingParticipant {
	merged := make([]meetingParticipant, 0, len(existing)+len(incoming))
	index := map[string]int{}
	add := func(item meetingParticipant) {
		key := meetingParticipantKey(item)
		if key == "" {
			return
		}
		if i, ok := index[key]; ok {
			merged[i] = mergeMeetingParticipant(merged[i], item)
			return
		}
		index[key] = len(merged)
		merged = append(merged, item)
	}
	for _, item := range existing {
		add(item)
	}
	for _, item := range incoming {
		add(item)
	}
	sort.SliceStable(merged, func(i, j int) bool {
		return strings.TrimSpace(merged[i].JoinedAt) < strings.TrimSpace(merged[j].JoinedAt)
	})
	return merged
}

func mergeMeetingParticipant(dst, src meetingParticipant) meetingParticipant {
	dst.ParticipantID = firstNonEmpty(dst.ParticipantID, src.ParticipantID)
	dst.LiveKitIdentity = firstNonEmpty(dst.LiveKitIdentity, src.LiveKitIdentity)
	dst.DisplayName = firstNonEmpty(src.DisplayName, dst.DisplayName)
	dst.Email = firstNonEmpty(dst.Email, src.Email)
	dst.JoinedAt = firstNonEmpty(dst.JoinedAt, src.JoinedAt)
	dst.LeftAt = firstNonEmpty(src.LeftAt, dst.LeftAt)
	dst.AudioTrackIDs = mergeStringSet(dst.AudioTrackIDs, src.AudioTrackIDs)
	return dst
}

func meetingParticipantKey(item meetingParticipant) string {
	for _, value := range []string{item.LiveKitIdentity, item.ParticipantID, item.Email, item.DisplayName} {
		if key := participantNameKey(value); key != "" {
			return key
		}
	}
	return ""
}

func mergeStringSet(left, right []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(left)+len(right))
	for _, value := range append(append([]string(nil), left...), right...) {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func participantDisplayNames(participants []meetingParticipant) string {
	names := make([]string, 0, len(participants))
	seen := map[string]struct{}{}
	for _, participant := range participants {
		name := strings.TrimSpace(firstNonEmpty(participant.DisplayName, participant.Email, participant.LiveKitIdentity, participant.ParticipantID))
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
	return strings.Join(names, ", ")
}

func meetingParticipantsFromNames(value string) []meetingParticipant {
	names := participantNamesFromText(value)
	items := make([]meetingParticipant, 0, len(names))
	for _, name := range names {
		items = append(items, meetingParticipant{
			ParticipantID:   speakerIDFromName(name),
			LiveKitIdentity: "",
			DisplayName:     name,
		})
	}
	return items
}

func segmentsFromTranscriptLines(lines []transcriptLine, defaultSource string, participants []meetingParticipant) []transcriptSegment {
	source := firstNonEmpty(defaultSource, speakerSourceUnknown)
	segments := make([]transcriptSegment, 0, len(lines))
	for i, line := range lines {
		lineSource := firstNonEmpty(line.Source, source)
		speakerName := normalizeSpeakerLabel(firstNonEmpty(line.SpeakerName, line.Speaker, "Speaker"))
		participant := matchParticipantForSpeaker(speakerName, participants)
		segment := transcriptSegment{
			ID:              firstNonEmpty(line.ID, fmt.Sprintf("seg-%d", i+1)),
			SpeakerID:       firstNonEmpty(line.SpeakerID, speakerIDFromName(speakerName)),
			SpeakerName:     speakerName,
			ParticipantID:   line.ParticipantID,
			LiveKitIdentity: line.LiveKitIdentity,
			TrackID:         line.TrackID,
			Source:          lineSource,
			Start:           line.Start,
			End:             line.End,
			Time:            line.Time,
			Text:            line.Text,
			Sentiment:       line.Sentiment,
		}
		if segment.ParticipantID == "" && segment.LiveKitIdentity == "" && sourceAllowsParticipantMatch(lineSource) && participant.ParticipantID != "" {
			segment.ParticipantID = participant.ParticipantID
			segment.LiveKitIdentity = firstNonEmpty(participant.LiveKitIdentity, participant.ParticipantID)
		}
		segments = append(segments, segment)
	}
	return segments
}

func sanitizeTranscriptSpeakerLabels(lines []transcriptLine, participants string) []transcriptLine {
	names := participantNamesFromText(participants)
	if len(lines) == 0 {
		return lines
	}
	out := make([]transcriptLine, len(lines))
	unknownToSpeaker := map[string]string{}
	nextSpeaker := nextFallbackSpeakerIndex(lines)
	if nextSpeaker <= 0 {
		nextSpeaker = 1
	}
	for i, line := range lines {
		speaker := normalizeSpeakerLabel(firstNonEmpty(line.SpeakerName, line.Speaker, "Speaker"))
		switch {
		case sourceAllowsParticipantMatch(line.Source):
			if known := matchKnownParticipantName(speaker, names); known != "" {
				speaker = known
			}
		case isGenericSpeakerName(speaker):
			// Keep neutral diarization/STT labels.
		case matchKnownParticipantName(speaker, names) != "":
			speaker = matchKnownParticipantName(speaker, names)
		default:
			key := participantNameKey(speaker)
			mapped := unknownToSpeaker[key]
			if mapped == "" {
				mapped = fmt.Sprintf("Speaker %d", nextSpeaker)
				nextSpeaker++
				unknownToSpeaker[key] = mapped
			}
			speaker = mapped
		}
		line.Speaker = speaker
		line.SpeakerName = speaker
		if line.SpeakerID == "" {
			line.SpeakerID = speakerIDFromName(speaker)
		}
		out[i] = line
	}
	return out
}

func transcriptSpeakersHaveUnsafeLabels(lines []transcriptLine, participants string) bool {
	names := participantNamesFromText(participants)
	for _, line := range lines {
		speaker := normalizeSpeakerLabel(firstNonEmpty(line.SpeakerName, line.Speaker))
		if speaker == "" || isGenericSpeakerName(speaker) || sourceAllowsParticipantMatch(line.Source) {
			continue
		}
		if matchKnownParticipantName(speaker, names) != "" {
			continue
		}
		return true
	}
	return false
}

func reportTranscriptsFromSegments(segments []transcriptSegment) []reportTranscript {
	items := make([]reportTranscript, 0, len(segments))
	for i, segment := range segments {
		speaker := firstNonEmpty(segment.SpeakerName, segment.SpeakerID, "Speaker")
		items = append(items, reportTranscript{
			ID:              firstNonEmpty(segment.ID, fmt.Sprintf("t%d", i+1)),
			Time:            segment.Time,
			Speaker:         speaker,
			SpeakerID:       segment.SpeakerID,
			SpeakerName:     segment.SpeakerName,
			ParticipantID:   segment.ParticipantID,
			LiveKitIdentity: segment.LiveKitIdentity,
			TrackID:         segment.TrackID,
			Source:          segment.Source,
			Start:           segment.Start,
			End:             segment.End,
			Text:            segment.Text,
			Sentiment:       segment.Sentiment,
		})
	}
	return items
}

func defaultSpeakerSource(lines []transcriptLine) string {
	counts := map[string]int{}
	for _, line := range lines {
		source := strings.TrimSpace(line.Source)
		if source != "" {
			counts[source]++
		}
	}
	bestSource := ""
	bestCount := 0
	for source, count := range counts {
		if count > bestCount {
			bestSource = source
			bestCount = count
		}
	}
	return firstNonEmpty(bestSource, speakerSourceSTT)
}

func reportTranscriptToSegments(items []reportTranscript, defaultSource string, participants []meetingParticipant) []transcriptSegment {
	lines := make([]transcriptLine, 0, len(items))
	for _, item := range items {
		lines = append(lines, transcriptLine{
			ID:              item.ID,
			Time:            item.Time,
			Speaker:         item.Speaker,
			SpeakerID:       item.SpeakerID,
			SpeakerName:     item.SpeakerName,
			ParticipantID:   item.ParticipantID,
			LiveKitIdentity: item.LiveKitIdentity,
			TrackID:         item.TrackID,
			Source:          item.Source,
			Start:           item.Start,
			End:             item.End,
			Text:            item.Text,
			Sentiment:       item.Sentiment,
		})
	}
	return segmentsFromTranscriptLines(lines, defaultSource, participants)
}

func defaultReportTranscriptSource(items []reportTranscript) string {
	counts := map[string]int{}
	for _, item := range items {
		if source := strings.TrimSpace(item.Source); source != "" {
			counts[source]++
		}
	}
	bestSource := ""
	bestCount := 0
	for source, count := range counts {
		if count > bestCount {
			bestSource = source
			bestCount = count
		}
	}
	return bestSource
}

func matchParticipantForSpeaker(speaker string, participants []meetingParticipant) meetingParticipant {
	speaker = strings.TrimSpace(speaker)
	if speaker == "" || isGenericSpeakerName(speaker) {
		return meetingParticipant{}
	}
	candidates := make([]string, 0, len(participants))
	for _, participant := range participants {
		candidates = append(candidates, participant.DisplayName)
	}
	if known := matchKnownParticipantName(speaker, candidates); known != "" {
		for _, participant := range participants {
			if participant.DisplayName == known {
				return participant
			}
		}
	}
	return meetingParticipant{}
}

func sourceAllowsParticipantMatch(source string) bool {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case speakerSourceParticipantTrack, speakerSourceManual:
		return true
	default:
		return false
	}
}

func speakerIDFromName(name string) string {
	name = normalizeSpeakerLabel(name)
	if name == "" {
		return ""
	}
	key := strings.ToLower(name)
	key = strings.ReplaceAll(key, " ", "-")
	return key
}
