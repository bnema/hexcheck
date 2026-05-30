.PHONY: test lint tidy check smoke-dumber

test:
	go test ./...

lint:
	golangci-lint run ./...

tidy:
	go mod tidy

check: tidy test

smoke-dumber:
	HEXCHECK_DUMBER_PATH=/home/brice/dev/projects/dumber go test ./test/smoke
