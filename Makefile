run: build
	@./bin/godis

cli:
	@go run ./cmd/cli/main.go

build:
	@go build -o bin/godis .
