# syntax=docker/dockerfile:1

FROM golang:1.21 as build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/*.go ./cmd/
COPY internal/*.go ./internal/

RUN CGO_ENABLED=0 GOOS=linux go build \
    -C cmd -o /metacritic-api -v

FROM scratch

COPY --from=build /metacritic-api /metacritic-api

CMD ["/metacritic-api"]