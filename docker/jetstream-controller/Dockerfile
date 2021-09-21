FROM golang:1.17 AS build
COPY . /go/src/nack
WORKDIR /go/src/nack
ARG VERSION
RUN VERSION=$VERSION make jetstream-controller.docker

FROM alpine:latest as osdeps
RUN apk add --no-cache ca-certificates

FROM scratch
COPY --from=build /go/src/nack/jetstream-controller.docker /jetstream-controller
COPY --from=osdeps /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

CMD ["/jetstream-controller"]
