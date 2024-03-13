.PHONY: fmt test build

# TODO(zeke): sometimes for npx to work, need to run `nvm use v20.10.0` first

fmt:
	go fmt ./...

test: fmt
	go test -v ./...

build: test
	env GOOS=linux CGO_ENABLED=0 go build -o bootstrap .

.PHONY: deploy config

deploy: build
	npx sls deploy

config:
	npx sls config credentials --provider aws --key <key> --secret <secret>