# syntax=docker/dockerfile:1

FROM golang:1.22 as build

ARG git_version=dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ ./cmd/
COPY internal/ ./internal/
# TODO(zeke): if we introduce pkg dir, need to copy it here

# -tags timetzdata is for embedding tz info for LoadLocation call in binary
RUN CGO_ENABLED=0 GOOS=linux GOEXPERIMENT=rangefunc go build \
     -tags=jsoniter -tags timetzdata -v -o /zest-api ./cmd/

FROM scratch

ARG git_version
ENV DD_ENV=$git_version
ENV GIT_SHA=$git_version

COPY --from=build /zest-api /zest-api

# need certificates else outgoing https requests fail
COPY --from=build etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENV GIN_MODE=release

CMD ["/zest-api", "server"]
