##################
## docker commands
##################
DOCKER := docker
ifeq ($(shell uname -s),Linux)
	DOCKER := sudo docker
endif

COMPOSE=$(DOCKER) compose

.PHONY: build up server dev monitor all clean down down-with-volumes build-with-version

clean:
	$(DOCKER) system prune -a

GIT_VERSION := $(shell git branch --show-current)
ifeq ($(GIT_VERSION),master)
	GIT_VERSION := $(shell git rev-parse --short HEAD)
endif


build-with-version:
	$(COMPOSE) --profile server build \
		--build-arg git_version=$(GIT_VERSION)

build:
	$(COMPOSE) --profile server build

up:
	$(COMPOSE) up -d

server: build
	$(COMPOSE) --profile server --profile dev up -d

dev:
	$(COMPOSE) --profile dev up -d

monitor: build
	$(COMPOSE) --profile monitoring --profile server up -d

COMPOSE_ALL_PROFILES=$(COMPOSE) --profile "*"

all: build
	$(COMPOSE_ALL_PROFILES) up -d

down:
	$(COMPOSE_ALL_PROFILES) down --remove-orphans

down-with-volumes:
	$(COMPOSE_ALL_PROFILES) down -v --remove-orphans


##################
## postgres commands
##################

# NOTE: it's honestly much easier to move things around with adminer...

.PHONY: dump restore

# took a while to figure out right configuration of dbname and username.
# this seemed to work!
dump:
	$(DOCKER) exec -it postgres pg_dump zest -U zeke > dump.sql

# trouble passing file, so doing `cat <file> | ... -f -`
# also trouble passing input through stdin so removed `-it`
# DIDN't WORK. Not sure why!! Trying with adminer instead.
# maybe there needs to be another way to "refresh" the db.
# can I just pull from the API to create resources locally?
restore:
	cat dump.sql | $(DOCKER) exec postgres psql -d zest -U zeke -f -

##################
## go tool commands
##################

GFLAGS=-tags=jsoniter
GVARS=GOEXPERIMENT=rangefunc
GORUN=$(GVARS) go run $(GFLAGS)

.PHONY: fmt run help test scrape go-dump backfill

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

go-dump:
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
