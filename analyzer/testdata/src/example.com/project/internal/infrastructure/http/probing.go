package http

import (
	"os"
	"strconv"
	"strings"
)

func DetectRuntimeThreads() int {
	value := strings.TrimSpace(os.Getenv("APP_RUNTIME_THREADS"))
	if value == "" {
		return 0
	}
	threads, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	if threads < 0 {
		return 0
	}
	if threads > 256 {
		return 256
	}
	return threads
}
