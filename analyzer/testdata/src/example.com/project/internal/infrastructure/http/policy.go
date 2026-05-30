package http

func DetectChanges(oldKeys, newKeys []string) string { // want "suspicious-business-logic-in-adapter"
	if len(oldKeys) == 0 {
		return "added"
	}
	if len(newKeys) == 0 {
		return "removed"
	}
	if len(oldKeys) > len(newKeys) {
		return "renamed"
	}
	return "unchanged"
}

func computeHistoryCandidateScore(hostMatch, recent bool, visits int) int { // want "suspicious-business-logic-in-adapter"
	score := 0
	if hostMatch {
		score += 50
	}
	if recent {
		score += 25
	}
	if visits > 10 {
		score += 5
	}
	if score > 100 {
		return 100
	}
	return score
}
