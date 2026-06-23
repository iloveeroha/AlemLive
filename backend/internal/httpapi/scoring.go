package httpapi

import (
	"math"
	"strings"
)

// speakerPositions holds the per-participant "Положения" metrics shown in
// the deep-dive view: an overall score, mood and engagement derived from
// their own lines, plus charisma/bias heuristics derived from how the
// conversation reacted around their turns.
type speakerPositions struct {
	Score       int
	Mood        int
	Engagement  int
	Charisma    int
	HasCharisma bool
	Bias        int
	HasBias     bool
}

type trendPoint struct {
	Time       string `json:"time"`
	Score      int    `json:"score"`
	Engagement int    `json:"engagement"`
	Mood       int    `json:"mood"`
}

// mediaPercentages holds the share of a participant's session spent with
// their microphone muted or camera off, derived from real device-state
// events reported by their own client during the call.
type mediaPercentages struct {
	MicMutedPercent  int `json:"micMutedPercent"`
	CameraOffPercent int `json:"cameraOffPercent"`
}

func clampScore(value int) int {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

// computeMeetingScores derives an overall score, engagement and mood (0-100)
// from real per-meeting signals: sentiment tags on transcript lines and how
// many distinct speakers participated. It is a deterministic heuristic, not
// a machine-learned model, but it varies with the actual meeting content
// instead of returning a constant placeholder.
func computeMeetingScores(lines []transcriptLine, speakerCount int) (score, engagement, mood int) {
	if len(lines) == 0 {
		return 75, 60, 75
	}

	positive, negative := 0, 0
	for _, line := range lines {
		switch line.Sentiment {
		case "positive":
			positive++
		case "negative":
			negative++
		}
	}

	moodBase := 75.0 + float64(positive)*3 - float64(negative)*5
	mood = clampScore(int(math.Round(moodBase)))

	engagementBase := 50.0 + math.Min(30, float64(speakerCount)*8) + math.Min(20, float64(len(lines))/3)
	engagement = clampScore(int(math.Round(engagementBase)))

	score = clampScore((mood + engagement) / 2)
	return score, engagement, mood
}

// computeSentimentTrend buckets the transcript into a handful of points across
// the meeting timeline so the UI can render a trend chart. Each bucket's mood
// and engagement are nudged from the meeting's baseline scores using the
// sentiment tags and talk density observed in that bucket.
func computeSentimentTrend(lines []transcriptLine, baseEngagement, baseMood int) []trendPoint {
	if len(lines) == 0 {
		return nil
	}

	bucketCount := len(lines) / 3
	if bucketCount < 4 {
		bucketCount = 4
	}
	if bucketCount > 12 {
		bucketCount = 12
	}
	if bucketCount > len(lines) {
		bucketCount = len(lines)
	}

	bucketSize := int(math.Ceil(float64(len(lines)) / float64(bucketCount)))
	points := make([]trendPoint, 0, bucketCount)

	for i := 0; i < len(lines); i += bucketSize {
		end := i + bucketSize
		if end > len(lines) {
			end = len(lines)
		}

		bucketPositive, bucketNegative := 0, 0
		for _, line := range lines[i:end] {
			switch line.Sentiment {
			case "positive":
				bucketPositive++
			case "negative":
				bucketNegative++
			}
		}

		moodDelta := (bucketPositive - bucketNegative) * 4
		mood := clampScore(baseMood + moodDelta)
		engagementDelta := (end - i - bucketSize/2) * 2
		engagement := clampScore(baseEngagement + engagementDelta)
		score := clampScore((mood + engagement) / 2)

		points = append(points, trendPoint{
			Time:       lines[i].Time,
			Score:      score,
			Engagement: engagement,
			Mood:       mood,
		})
	}

	return points
}

// computeParticipantPositions derives per-participant "Положения" metrics
// from the transcript: mood/engagement from this speaker's own lines and
// talk share, plus charisma (how others reacted right after this person
// spoke) and bias (how this person reacted right after someone else spoke).
// Charisma/bias are only reported when the transcript actually contains a
// qualifying speaker transition.
func computeParticipantPositions(name string, talkPercent int, lines []transcriptLine) speakerPositions {
	isSameSpeaker := func(speaker string) bool {
		return strings.EqualFold(strings.TrimSpace(speaker), strings.TrimSpace(name))
	}

	ownPositive, ownNegative, ownLines := 0, 0, 0
	reactionPositive, reactionNegative, reactionLines := 0, 0, 0
	biasPositive, biasNegative, biasLines := 0, 0, 0

	for i, line := range lines {
		own := isSameSpeaker(line.Speaker)
		previousBySameSpeaker := i > 0 && isSameSpeaker(lines[i-1].Speaker)

		if own {
			ownLines++
			switch line.Sentiment {
			case "positive":
				ownPositive++
			case "negative":
				ownNegative++
			}
			if i > 0 && !previousBySameSpeaker {
				biasLines++
				switch line.Sentiment {
				case "positive":
					biasPositive++
				case "negative":
					biasNegative++
				}
			}
			continue
		}

		if previousBySameSpeaker {
			reactionLines++
			switch line.Sentiment {
			case "positive":
				reactionPositive++
			case "negative":
				reactionNegative++
			}
		}
	}

	mood := clampScore(75 + ownPositive*4 - ownNegative*6)
	engagement := clampScore(50 + talkPercent/2 + ownLines*2)
	score := clampScore((mood + engagement) / 2)

	positions := speakerPositions{Score: score, Mood: mood, Engagement: engagement}
	if reactionLines > 0 {
		positions.HasCharisma = true
		positions.Charisma = clampScore(75 + (reactionPositive-reactionNegative)*8)
	}
	if biasLines > 0 {
		positions.HasBias = true
		positions.Bias = clampScore(75 + (biasPositive-biasNegative)*8)
	}

	return positions
}
