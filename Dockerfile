FROM golang:1.14.12-alpine AS builder

ARG BUILD_VERSION=0.0.0
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG PROGNAME=nginx-mail-auth-http-server
ARG LISTEN_ADDRESS="127.0.0.1"
ARG LISTEN_PORT="8080"

RUN mkdir -p -v /src
WORKDIR /src
ADD . /src

RUN GOOS="${TARGETOS}" GOARCH="${TARGETARCH}" go build -ldflags="-X 'main.BuildVersion=${BUILD_VERSION}'" -v -o nginx-mail-auth-http-server .


FROM alpine:3.13

COPY --from=builder /src/nginx-mail-auth-http-server nginx-mail-auth-http-server

ENTRYPOINT ["./nginx-mail-auth-http-server"]
