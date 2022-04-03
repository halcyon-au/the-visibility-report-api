#!/bin/sh

if [ $1 = "worker" ]; then
  cd ./cmd/worker
  go build .
  ./worker
else
  cd ./cmd/api
  go build .
  ./api
fi