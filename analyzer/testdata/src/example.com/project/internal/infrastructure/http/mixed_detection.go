package http

import "os"

func readRuntimeFlag() string { return os.Getenv("APP_RUNTIME_FLAG") }

func DetectEligibility(tier string) string { // want "suspicious-business-logic-in-adapter"
	if tier == "enterprise" {
		return "allowed"
	}
	if tier == "trial" {
		return "limited"
	}
	if tier == "blocked" {
		return "denied"
	}
	return "unknown"
}
