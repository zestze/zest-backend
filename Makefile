##################
## docker commands
##################
DOCKER := docker
ifeq ($(shell uname -s),Linux)
	DOCKER := sudo docker
endif

COMPOSE=$(DOCKER) compose

.PHONY: build up server dev monitor all clean down down-with-volumes build-with-version

# TODO(zeke): make this grab branch name if not on master / main
build-with-version:
	$(COMPOSE) --profile server build \
		--build-arg git_version=$(shell git rev-parse --short HEAD)

build:
	$(COMPOSE) --profile server build

up:
	$(COMPOSE) up -d

server: build
	$(COMPOSE) --profile server up -d

dev:
	$(COMPOSE) --profile dev up -d

monitor: build
	$(COMPOSE) --profile monitoring --profile server up -d

all: build
	$(COMPOSE) --profile "*" up -d

clean:
	$(DOCKER) system prune -a

down:
	$(COMPOSE) --profile "*" down --remove-orphans

down-with-volumes:
	$(COMPOSE) --profile "*" down -v --remove-orphans

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

# TODO(zeke): use the running container, but add a new database?
test-db: fmt
	go test -short -tags=integration ./internal/metacritic
	#atlas schema clean -u "postgres://zeke:reyna@localhost:5432/integration?sslmode=disable" --auto-approve
	#atlas schema apply -u "postgres://zeke:reyna@localhost:5432/integration?sslmode=disable" --to file://schema.sql \
	#	--auto-approve \
	#	--dev-url "postgres://atlas:pass@localhost:5444/postgres?sslmode=disable"
#	# spin up postgres db
#	docker run --name postgres-integration -e POSTGRES_USER=zeke -e POSTGRES_PASSWORD=reyna -e POSTGRES_DB=integration

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

deploy: build-with-version
	$(DOCKER) save zest-backend-zest-api > zest-api.tar
	scp zest-api.tar droplet:~/workspace/zest-api.tar
	ssh droplet 'make -C workspace deploy'


serverless:
	doctl serverless deploy serverless/digitalocean
	# can also test with doctl serverless functions invoke zest/refresh
