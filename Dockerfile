FROM golang:1.18.1-alpine AS builder

ARG BUILD_VERSION=0.0.0
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG PROGNAME=nginx-mail-auth-http-server
ARG LISTEN_ADDRESS="127.0.0.1"
ARG LISTEN_PORT="8080"

RUN mkdir -p -v /src
WORKDIR /src
COPY . /src

RUN apk add git
RUN GOOS="${TARGETOS}" GOARCH="${TARGETARCH}" go build -ldflags="-X 'main.BuildVersion=${BUILD_VERSION}'" -v -o nginx-mail-auth-http-server .


FROM alpine:3.16

COPY --from=builder /src/nginx-mail-auth-http-server nginx-mail-auth-http-server

ENTRYPOINT ["./nginx-mail-auth-http-server"]
