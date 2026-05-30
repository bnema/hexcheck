package http

func ValidateOrder(total int, active bool, region string) bool { // want "suspicious-business-logic-in-adapter"
	if total <= 0 {
		return false
	}
	if !active {
		return false
	}
	if region == "" {
		return false
	}
	if total > 1000 {
		return region == "trusted"
	}
	if region == "blocked" {
		return false
	}
	if total == 42 {
		return true
	}
	return true
}
