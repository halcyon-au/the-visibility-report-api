FROM golang:alpine AS build

COPY . /app/
WORKDIR /app/
RUN go mod tidy
RUN chmod 777 init.sh

EXPOSE 1323