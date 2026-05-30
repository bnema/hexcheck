.PHONY: test lint tidy check smoke-local

test:
	go test ./...

lint:
	golangci-lint run ./...

tidy:
	go mod tidy

check: tidy test

smoke-local:
	@if [ -z "$(HEXCHECK_SMOKE_REPO)" ]; then \
		echo "HEXCHECK_SMOKE_REPO must point to a local Go repository for smoke testing"; \
		exit 1; \
	fi
	go test ./test/smoke
