FROM golang:1.19-alpine as builder

RUN apk add --no-cache gcc g++ bash git

WORKDIR $GOPATH/src/github.com/toni-moreno/oracle_collector

COPY go.mod go.sum ./

COPY pkg pkg
COPY .git .git
COPY build.go ./

RUN go run build.go  build

FROM alpine:latest
MAINTAINER Toni Moreno <toni.moreno@gmail.com>



VOLUME ["/opt/oracle_collector/conf", "/opt/oracle_collector/log"]

EXPOSE 8090

COPY --from=builder /go/src/github.com/toni-moreno/oracle_collector/bin/oracle_collector ./bin/

WORKDIR /opt/oracle_collector
COPY ./conf/sample.oracle_collector.toml ./conf/oracle_collector.toml

ENTRYPOINT ["/bin/oracle_collector"]
