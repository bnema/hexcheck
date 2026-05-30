package http

import "example.com/project/internal/domain"

func computeUserScore(user *domain.User) int { // want "suspicious-business-logic-in-adapter"
	score := 0
	if user.Name == "admin" {
		score += 100
	}
	if stringsHasPrefix(user.Name, "power") {
		score += 50
	}
	if user.Name == "guest" {
		score -= 10
	}
	if score > 100 {
		return 100
	}
	return score
}

func stringsHasPrefix(value, prefix string) bool {
	return len(value) >= len(prefix) && value[:len(prefix)] == prefix
}
