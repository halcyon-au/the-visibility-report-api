FROM golang:alpine AS build

COPY . /app/
WORKDIR /app/
RUN go mod tidy

EXPOSE 1323