# syntax=docker/dockerfile:1

FROM golang:1.21 as build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/*.go ./cmd/
COPY internal/reddit/*.go ./internal/reddit/
COPY internal/metacritic/*.go ./internal/metacritic/

RUN CGO_ENABLED=0 GOOS=linux go build \
    -C cmd -o /zest-api -v

FROM scratch

COPY --from=build /zest-api /zest-api

CMD ["/zest-api"]