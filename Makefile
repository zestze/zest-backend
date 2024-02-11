.PHONY: run fmt test deploy scrape serverless help up build clean down

GFLAGS=-tags=jsoniter

##################
## docker commands
##################

build:
	sudo docker compose build

up: build
	sudo docker compose up -d

clean:
	sudo docker system prune -a

down:
	sudo docker compose down

##################
## go tool commands
##################

run:
	go run $(GFLAGS) ./cmd server

help:
	go run $(GFLAGS) ./cmd --help

fmt:
	go mod tidy
	go fmt ./...
	go vet ./...

test:
	go test -short ./...

scrape:
	go run $(GFLAGS) ./cmd scrape reddit

dump:
	go run $(GFLAGS) ./cmd dump

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
