FROM alpine:3.6

COPY deploy/_output/envconsul /envconsul
COPY deploy/_output/protor/bin/protor /protor

EXPOSE 9090

ENTRYPOINT ["/protor"]
