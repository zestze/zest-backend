# syntax=docker/dockerfile:1

FROM golang:1.21 as build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ ./cmd/
COPY internal/ ./internal/
# TODO(zeke): if we introduce pkg dir, need to copy it here

RUN CGO_ENABLED=0 GOOS=linux go build \
     -tags=jsoniter -v -o /zest-api ./cmd/

FROM scratch

COPY --from=build /zest-api /zest-api

# need certificates else outgoing https requests fail
COPY --from=build etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY migrations/*sql ./migrations/

ENV GIN_MODE=release

CMD ["/zest-api", "server"]
