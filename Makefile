##################
## docker commands
##################
#DOCKER=sudo docker
DOCKER=docker
ifeq ($(shell uname -s),Linux)
	DOCKER := sudo docker
endif
COMPOSE=$(DOCKER) compose

.PHONY: build up up-debug up-monitoring clean down down-with-volumes

build:
	$(COMPOSE) build

up: build
	$(COMPOSE) up -d

up-debug: build
	$(COMPOSE) --profile debug up -d

up-monitoring: build
	$(COMPOSE) --profile monitoring up -d

clean:
	$(DOCKER) system prune -a

down:
	$(COMPOSE) --profile monitoring --profile debug down

down-with-volumes:
	$(COMPOSE) down -v

##################
## go tool commands
##################

GFLAGS=-tags=jsoniter
GVARS=GOEXPERIMENT=rangefunc
GORUN=$(GVARS) go run $(GFLAGS)

.PHONY: fmt run help test scrape dump backfill

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

backfill:
	CREDS=--username=$ZEST_USERNAME --password=$ZEST_PASSWORD
	#$(GORUN) ./cmd backfill --help
	#$(GORUN) ./cmd backfill --resource=reddit $(CREDS)
	$(GORUN) ./cmd backfill --resource=spotify $(CREDS) \
		--start=2024-04-04 --end=2024-05-28

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
