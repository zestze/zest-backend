.PHONY: run fmt test

run:
	go run ./cmd

fmt:
	go fmt ./...
	go vet ./...

test:
	go test ./...
