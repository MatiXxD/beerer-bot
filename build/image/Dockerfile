FROM golang:1.26.5-alpine AS builder

ENV CGO_ENABLED=0
ENV GOOS=linux

RUN apk update --no-cache && apk add --no-cache tzdata

COPY go.mod go.sum /beerer-bot/

WORKDIR /beerer-bot

RUN go mod download
COPY . /beerer-bot
RUN go build -ldflags="-s -w" -o /beerer-bot/bin/beerer-bot ./cmd/bot/main.go

FROM alpine:latest

ENV TZ=Europe/Moscow

EXPOSE 8080

WORKDIR /beerer-bot

COPY --from=builder /usr/share/zoneinfo/${TZ} /etc/localtime
RUN echo $TZ > /etc/timezone

COPY --from=builder /beerer-bot/bin/beerer-bot .