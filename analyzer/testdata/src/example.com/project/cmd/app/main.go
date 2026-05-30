package main

import "example.com/project/internal/domain"

func ResolveStartupMode(user *domain.User) string { // want "suspicious-business-logic-in-adapter: entrypoint function ResolveStartupMode"
	user.Name = "seen"
	if user.Name == "admin" {
		return "admin"
	}
	if user.Name == "guest" {
		return "guest"
	}
	if user.Name == "" {
		return "empty"
	}
	return "normal"
}
