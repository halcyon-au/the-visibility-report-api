#!/usr/bin/env sh
swag fmt -d ./cmd/api,./controllers
swag init -d ./cmd/api,./controllers -o cmd/api/docs
