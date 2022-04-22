#!/bin/sh
if [ "$1" = "worker" ]; then
  cd ./cmd/worker
  go build -buildvcs=false .
  ./worker
else
  cd ./cmd/api
  go build -buildvcs=false .
  ./api
fi
