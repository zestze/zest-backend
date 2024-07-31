##################
## docker commands
##################
DOCKER=sudo docker
COMPOSE=$(DOCKER) compose

.PHONY: build up dev up-monitoring clean down down-with-volumes

build:
	$(COMPOSE) build

up: build
	$(COMPOSE) up -d

dev: build
	$(COMPOSE) --profile dev up -d

up-monitoring: build
	$(COMPOSE) --profile monitoring up -d

clean:
	$(DOCKER) system prune -a

down:
	$(COMPOSE) --profile monitoring --profile dev down --remove-orphans

down-with-volumes:
	$(COMPOSE) down -v

##################
## go tool commands
##################

GFLAGS=-tags=jsoniter
GVARS=GOEXPERIMENT=rangefunc
GORUN=$(GVARS) go run $(GFLAGS)

.PHONY: fmt run help test scrape dump

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

.PHONY: deploy serverless

deploy: build
	$(DOCKER) save zest-backend-zest-api > zest-api.tar
	scp zest-api.tar droplet:~/workspace/zest-api.tar
	ssh droplet 'make -C workspace deploy'


serverless:
	doctl serverless deploy serverless/digitalocean
	# can also test with doctl serverless functions invoke zest/refresh
