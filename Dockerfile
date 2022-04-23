FROM golang:alpine AS build

COPY . /app/
WORKDIR /app/

RUN apk add git
RUN go mod tidy
RUN chmod 777 start.sh

EXPOSE 1323
