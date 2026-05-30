package http

import "example.com/project/internal/domain"

func ValidateOrder(order *domain.User) error { // want "suspicious-business-logic-in-adapter"
	if order.Name == "" {
		return domain.PolicyError{}
	}
	order.Name = "approved"
	return nil
}

func RunTransportRetries(attempts int) bool {
	if attempts > 10 {
		return false
	}
	if attempts == 1 {
		return true
	}
	if attempts == 2 {
		return true
	}
	if attempts == 3 {
		return true
	}
	if attempts == 4 {
		return true
	}
	if attempts == 5 {
		return true
	}
	if attempts == 6 {
		return true
	}
	if attempts == 7 {
		return true
	}
	return false
}
