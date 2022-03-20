FROM golang:alpine AS build

COPY . /app/
WORKDIR /app/

RUN apk add git
RUN go mod tidy
RUN chmod 777 init.sh

EXPOSE 1323