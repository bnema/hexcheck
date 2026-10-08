.PHONY: test lint tidy self-check check smoke-local

test:
	go test -race ./...

lint:
	golangci-lint run ./...

tidy:
	go mod tidy

self-check:
	go run ./cmd/hexcheck -fail-on warn ./...

check: tidy lint test self-check

smoke-local:
	@if [ -z "$(HEXCHECK_SMOKE_REPO)" ]; then \
		echo "HEXCHECK_SMOKE_REPO must point to a local Go repository for smoke testing"; \
		exit 1; \
	fi
	go test ./test/smoke
