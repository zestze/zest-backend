.PHONY: run fmt test prometheus deploy scrape serverless help

run:
	go run -tags=jsoniter ./cmd server

help:
	go run ./cmd --help

fmt:
	go mod tidy
	go fmt ./...
	go vet ./...

test:
	go test ./...

prometheus:
	# create volume separately
	sudo docker run \
		-v prometheus-data:/prometheus \
		--network="host" \
		-v ${CURDIR}/prometheus.yml:/etc/prometheus/prometheus.yml \
		prom/prometheus

	#-p 9090:9090 \
	#--add-host host.docker.internal:host-gateway \

deploy:
	sudo docker compose build
	sudo docker save zest-backend-zest-api > zest-api.tar
	scp zest-api.tar droplet:~/workspace/zest-api.tar
	ssh droplet 'make -C workspace deploy'

scrape:
	go run ./cmd scrape reddit

serverless:
	doctl serverless deploy serverless
	# can also test with doctl serverless functions invoke zest/refresh
