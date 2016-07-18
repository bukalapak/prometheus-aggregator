FROM golang:1-onbuild
MAINTAINER Adam Szpakowski <adam@szpakowski.info>
MAINTAINER Roland Rifandi Utama <roland_hawk@yahoo.com>

# ingress: samples via UDP
EXPOSE 8080/udp
# egress: metrics for prometheus to scrape
EXPOSE 9090
