package http

func ValidateOrder(total int, active bool) bool { // want "suspicious-business-logic-in-adapter"
	if total <= 0 {
		return false
	}
	if !active {
		return false
	}
	return true
}
