FROM golang:1.15 AS build
COPY . /go/src/nack
WORKDIR /go/src/nack
RUN make nats-server-config-reloader.docker

FROM alpine:latest as osdeps
RUN apk add --no-cache ca-certificates

FROM alpine:3.8
COPY --from=build /go/src/nack/nats-server-config-reloader.docker /usr/local/bin/nats-server-config-reloader
COPY --from=osdeps /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

CMD ["nats-server-config-reloader"]
