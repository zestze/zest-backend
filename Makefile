.PHONY: run fmt test deploy scrape serverless help up build clean down up-debug down-with-volumes

##################
## docker commands
##################
DOCKER=sudo docker
COMPOSE=$(DOCKER) compose

build:
	$(COMPOSE) build

up: build
	$(COMPOSE) up -d

up-debug: build
	$(COMPOSE) --profile debug up -d

clean:
	$(DOCKER) system prune -a

down:
	$(COMPOSE) down

down-with-volumes:
	$(COMPOSE) down -v

##################
## go tool commands
##################

GFLAGS=-tags=jsoniter
GVARS=GOEXPERIMENT=rangefunc
GORUN=$(GVARS) go run $(GFLAGS)

fmt:
	go mod tidy
	go fmt ./...
	go vet ./...

run: fmt
	$(GORUN) ./cmd server

help: fmt
	$(GORUN) ./cmd --help

test: fmt
	go test -short ./...

scrape:
	$(GORUN) ./cmd scrape reddit

dump:
	$(GORUN) ./cmd dump

##################
## deploy commands
##################

deploy: build
	$(DOCKER) save zest-backend-zest-api > zest-api.tar
	scp zest-api.tar droplet:~/workspace/zest-api.tar
	ssh droplet 'make -C workspace deploy'


serverless:
	doctl serverless deploy serverless
	# can also test with doctl serverless functions invoke zest/refresh
