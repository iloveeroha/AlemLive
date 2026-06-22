package httpapi

import (
	"sort"
	"strings"
	"unicode"
)

var keywordStopWords = map[string]struct{}{
	"и": {}, "в": {}, "не": {}, "что": {}, "как": {}, "это": {}, "на": {}, "с": {}, "со": {},
	"по": {}, "для": {}, "или": {}, "его": {}, "ее": {}, "их": {}, "она": {}, "он": {}, "оно": {},
	"мы": {}, "вы": {}, "ты": {}, "я": {}, "же": {}, "бы": {}, "ли": {}, "то": {}, "так": {},
	"вот": {}, "ну": {}, "да": {}, "нет": {}, "очень": {}, "просто": {}, "можно": {}, "нужно": {},
	"есть": {}, "был": {}, "была": {}, "было": {}, "были": {}, "будет": {}, "будем": {}, "вообще": {},
	"еще": {}, "уже": {}, "если": {}, "когда": {}, "тут": {}, "там": {}, "тоже": {}, "только": {},
	"чтобы": {}, "которые": {}, "который": {}, "которая": {}, "которое": {}, "этого": {}, "этой": {},
	"этому": {}, "этим": {}, "себя": {}, "себе": {}, "нам": {}, "вам": {}, "им": {}, "при": {},
	"за": {}, "из": {}, "от": {}, "до": {}, "под": {}, "над": {}, "без": {}, "про": {}, "всё": {},
	"все": {}, "всех": {}, "всем": {}, "кто": {}, "чем": {}, "чём": {}, "более": {}, "менее": {},
	"the": {}, "and": {}, "for": {}, "are": {}, "but": {}, "not": {}, "you": {}, "with": {}, "this": {},
	"that": {}, "have": {}, "from": {}, "your": {}, "all": {}, "can": {}, "will": {}, "just": {},
	"about": {}, "into": {}, "over": {}, "then": {}, "than": {}, "what": {}, "when": {}, "there": {},
	"their": {}, "they": {}, "them": {}, "some": {}, "like": {}, "going": {}, "want": {}, "know": {},
}

var sentimentPositiveWords = []string{
	"отлично", "супер", "круто", "хорошо", "удобно", "легко", "нравится", "спасибо", "класс",
	"здорово", "успешно", "получилось", "рады", "приятно", "great", "awesome", "love", "easy", "thanks",
}

var sentimentNegativeWords = []string{
	"проблема", "сложно", "ошибка", "неудобно", "плохо", "риск", "не получается", "не работает",
	"баг", "сломалось", "беспокоит", "issue", "problem", "bug", "broken", "hard", "difficult",
}

func extractKeywords(text string, limit int) []string {
	counts := map[string]int{}
	order := make([]string, 0)

	var b strings.Builder
	flush := func() {
		word := strings.ToLower(b.String())
		b.Reset()
		if len([]rune(word)) < 4 {
			return
		}
		if _, stop := keywordStopWords[word]; stop {
			return
		}
		if _, seen := counts[word]; !seen {
			order = append(order, word)
		}
		counts[word]++
	}
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			flush()
		}
	}
	flush()

	sort.SliceStable(order, func(i, j int) bool {
		return counts[order[i]] > counts[order[j]]
	})

	if limit <= 0 || limit > len(order) {
		limit = len(order)
	}
	return order[:limit]
}

func detectSentiment(text string) string {
	lower := strings.ToLower(text)
	for _, word := range sentimentNegativeWords {
		if strings.Contains(lower, word) {
			return "negative"
		}
	}
	for _, word := range sentimentPositiveWords {
		if strings.Contains(lower, word) {
			return "positive"
		}
	}
	return ""
}

func annotateTranscriptSentiment(lines []transcriptLine) []transcriptLine {
	for i := range lines {
		lines[i].Sentiment = detectSentiment(lines[i].Text)
	}
	return lines
}
