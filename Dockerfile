# syntax=docker/dockerfile:experimental

FROM golang:1.14-alpine AS builder

ENV deps "git"

RUN apk update && apk upgrade

RUN apk add --no-cache $deps

ENV CGO_ENABLED 0

WORKDIR /build/

COPY go.mod go.sum /build/
RUN --mount=type=cache,target=/root/go/pkg/mod go mod download

RUN apk del --purge $deps

COPY cmd /build/cmd
COPY pkg /build/pkg
RUN --mount=type=cache,target=/root/.cache/go-build go build -trimpath -o /usr/local/bin/main -ldflags="-s -w" /build/cmd/main.go

FROM gcr.io/distroless/base
COPY --from=builder /usr/local/bin/main /usr/local/bin/main

ENTRYPOINT ["/usr/local/bin/main"]
CMD ["server"]