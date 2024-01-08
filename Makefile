.PHONY: run fmt test prometheus deploy scrape serverless

run:
	go run ./cmd

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

scrape:
	go run ./cmd -f

serverless:
	doctl serverless deploy serverless
	# can also test with doctl serverless functions invoke zest/refresh
