.PHONY: test lint build bench netcheck

test:
	go test -race ./...

lint:
	golangci-lint run

netcheck:
	pwsh -NoProfile -File scripts/check_net.ps1

build:
	pwsh -NoProfile -File scripts/build.ps1

bench:
	go test -bench=. -benchmem ./internal/index ./internal/search ./internal/parser
