package search

import (
	"strings"
	"unicode"
)

const maxQueryCandidates = 6

var queryStopWords = map[string]struct{}{
	"a": {}, "an": {}, "and": {}, "are": {}, "about": {}, "by": {}, "call": {}, "called": {}, "calls": {}, "does": {}, "for": {},
	"find": {}, "from": {}, "how": {}, "in": {}, "is": {}, "me": {}, "of": {}, "on": {},
	"or": {}, "show": {}, "tell": {}, "the": {}, "to": {}, "use": {}, "using": {}, "what": {},
	"where": {}, "which": {}, "why": {}, "with": {},
}

// QueryCandidates turns a natural-language investigation question into a small,
// ordered set of likely repository terms. It preserves explicit identifiers and
// derives compound candidates from the meaningful words in a question.
func QueryCandidates(query string) []string {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}

	seen := make(map[string]struct{})
	var candidates []string
	add := func(candidate string) {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			return
		}
		key := strings.ToLower(candidate)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		candidates = append(candidates, candidate)
	}

	words := queryWords(query)
	for _, token := range rawQueryTokens(query) {
		if containsUppercase(token) {
			add(token)
		}
	}
	if !strings.ContainsAny(query, " \t\r\n") {
		add(query)
	}

	var meaningful []string
	for _, word := range words {
		if _, stop := queryStopWords[strings.ToLower(word)]; stop || len([]rune(word)) < 3 {
			continue
		}
		meaningful = append(meaningful, word)
	}
	for i := 0; i+1 < len(meaningful); i++ {
		add(pascalCase(meaningful[i] + " " + meaningful[i+1]))
	}
	if len(meaningful) > 1 {
		add(pascalCase(strings.Join(meaningful, " ")))
	}
	if len(meaningful) == 1 {
		add(meaningful[0])
	}
	if len(candidates) == 0 {
		add(query)
	}
	if len(candidates) > maxQueryCandidates {
		return candidates[:maxQueryCandidates]
	}
	return candidates
}

func queryWords(query string) []string {
	var words []string
	var current []rune
	runes := []rune(query)
	flush := func() {
		if len(current) > 0 {
			words = append(words, string(current))
			current = nil
		}
	}
	for i, r := range runes {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			flush()
			continue
		}
		if len(current) > 0 && unicode.IsUpper(r) && (unicode.IsLower(current[len(current)-1]) || (i+1 < len(runes) && unicode.IsLower(runes[i+1]))) {
			flush()
		}
		current = append(current, r)
	}
	flush()
	return words
}

func rawQueryTokens(query string) []string {
	var tokens []string
	var current []rune
	flush := func() {
		if len(current) > 0 {
			tokens = append(tokens, string(current))
			current = nil
		}
	}
	for _, r := range []rune(query) {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			flush()
			continue
		}
		current = append(current, r)
	}
	flush()
	return tokens
}

func pascalCase(value string) string {
	words := queryWords(value)
	for i, word := range words {
		runes := []rune(strings.ToLower(word))
		if len(runes) > 0 {
			runes[0] = unicode.ToUpper(runes[0])
		}
		words[i] = string(runes)
	}
	return strings.Join(words, "")
}

func containsUppercase(value string) bool {
	for _, r := range value {
		if unicode.IsUpper(r) {
			return true
		}
	}
	return false
}
