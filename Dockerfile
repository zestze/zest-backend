# syntax=docker/dockerfile:1

FROM golang:1.21 as build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/*.go ./cmd/
COPY internal/reddit/*.go ./internal/reddit/
COPY internal/metacritic/*.go ./internal/metacritic/
COPY internal/requestid/*.go ./internal/requestid/
COPY internal/zlog/*.go ./internal/zlog/

RUN CGO_ENABLED=0 GOOS=linux go build \
     -v -o /zest-api ./cmd/

FROM scratch

COPY --from=build /zest-api /zest-api

# need certificates else outgoing https requests fail
COPY --from=build etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

CMD ["/zest-api"]
