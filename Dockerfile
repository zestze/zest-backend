# syntax=docker/dockerfile:1

FROM golang:1.21 as build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

# TODO(zeke): maybe clean this up by introducing .dockerignore and .gitignore?
COPY cmd/*.go ./cmd/
COPY internal/reddit/*.go ./internal/reddit/
COPY internal/metacritic/*.go ./internal/metacritic/
COPY internal/requestid/*.go ./internal/requestid/
COPY internal/zlog/*.go ./internal/zlog/
COPY internal/ztrace/*.go ./internal/ztrace/
COPY internal/zql/*.go ./internal/zql/
COPY internal/user/*.go ./internal/user/

RUN CGO_ENABLED=0 GOOS=linux go build \
     -tags=jsoniter -v -o /zest-api ./cmd/

FROM scratch

COPY --from=build /zest-api /zest-api

# need certificates else outgoing https requests fail
COPY --from=build etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENV GIN_MODE=release

CMD ["/zest-api", "server"]
