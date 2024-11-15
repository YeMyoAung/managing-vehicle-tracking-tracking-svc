run:
	@go mod tidy && go run main.go

build:
	@go mod tidy && go build -o bin/tracking-svc

test:
	@go mod tidy && go test -v -cover -race ./...

.PHONY: run build test