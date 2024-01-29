.PHONY: run fmt test deploy scrape serverless help up build clean down

##################
## docker commands
##################

build:
	sudo docker compose --profile monitoring build

up: build
	sudo docker compose --profile monitoring up -d

clean:
	sudo docker system prune -a

down:
	sudo docker compose --profile monitoring down

##################
## go tool commands
##################

run:
	go run -tags=jsoniter ./cmd server

help:
	go run ./cmd --help

fmt:
	go mod tidy
	go fmt ./...
	go vet ./...

test:
	go test -short ./...

scrape:
	go run ./cmd scrape reddit

##################
## deploy commands
##################

deploy: build
	sudo docker save zest-backend-zest-api > zest-api.tar
	scp zest-api.tar droplet:~/workspace/zest-api.tar
	ssh droplet 'make -C workspace deploy'


serverless:
	doctl serverless deploy serverless
	# can also test with doctl serverless functions invoke zest/refresh
