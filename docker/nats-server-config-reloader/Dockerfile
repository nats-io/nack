FROM golang:1.17 AS build
COPY . /go/src/nack
WORKDIR /go/src/nack
ARG VERSION
RUN VERSION=$VERSION make nats-server-config-reloader.docker

FROM alpine:latest as osdeps
RUN apk add --no-cache ca-certificates

FROM alpine:3.15
COPY --from=build /go/src/nack/nats-server-config-reloader.docker /usr/local/bin/nats-server-config-reloader
COPY --from=osdeps /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

CMD ["nats-server-config-reloader"]
