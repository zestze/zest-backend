.PHONY: run fmt test prometheus

run:
	go run ./cmd

fmt:
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