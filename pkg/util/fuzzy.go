package util

import "strings"

func FuzzyScore(query, target string) int {
	query = strings.ToLower(strings.TrimSpace(query))
	target = strings.ToLower(strings.TrimSpace(target))
	if query == "" {
		return 1
	}
	if target == query {
		return 1000
	}
	if strings.Contains(target, query) {
		return 700 - (len(target) - len(query))
	}
	score := 0
	cursor := 0
	lastMatch := -2
	for _, r := range query {
		found := false
		for cursor < len(target) {
			if rune(target[cursor]) == r {
				score += 25
				if lastMatch+1 == cursor {
					score += 15
				}
				if cursor == 0 || target[cursor-1] == ' ' || target[cursor-1] == '-' || target[cursor-1] == '_' {
					score += 20
				}
				lastMatch = cursor
				cursor++
				found = true
				break
			}
			cursor++
		}
		if !found {
			return 0
		}
	}
	return score - (len(target) - len(query))
}

func FuzzyMatch(query, target string) bool {
	return FuzzyScore(query, target) > 0
}
