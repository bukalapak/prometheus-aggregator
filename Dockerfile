FROM alpine:3.1
MAINTAINER Roland Rifandi Utama <roland_hawk@yahoo.com>

WORKDIR /app
EXPOSE 8080/udp 9090

COPY ./deploy/_output/prometheus-aggregator /app/

ENTRYPOINT ["/app/prometheus-aggregator"]
