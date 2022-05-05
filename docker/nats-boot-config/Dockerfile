FROM golang:1.17 AS build
COPY . /go/src/nack
WORKDIR /go/src/nack
ARG VERSION
RUN VERSION=$VERSION make nats-boot-config.docker

FROM alpine:latest as osdeps
RUN apk add --no-cache ca-certificates

FROM alpine:3.8
COPY --from=build /go/src/nack/nats-boot-config.docker /usr/local/bin/nats-boot-config
COPY --from=osdeps /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
RUN ln -s /usr/local/bin/nats-boot-config /usr/local/bin/nats-pod-bootconfig

CMD ["nats-boot-config"]
