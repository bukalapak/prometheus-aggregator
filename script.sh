#! /bin/bash

version=`cat VERSION`

function docker-build() {
  sudo docker build -t $1 $2
  sudo docker tag $1 $1:$version
}

function compile() {
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build
}

function build() {
  compile
  docker-build rolandhawk/prometheus-aggregator .
}

function docker-push() {
  sudo docker push $1:latest
  sudo docker push $1:$version
}

function push() {
  docker-push rolandhawk/prometheus-aggregator
}

eval $1
