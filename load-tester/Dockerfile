# syntax=docker/dockerfile:1

FROM golang:1.25.4-alpine AS builder
WORKDIR /src
RUN apk add --no-cache ca-certificates tzdata
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/load-tester ./

FROM alpine:3.23
RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -u 10001 app
USER app
WORKDIR /home/app
COPY --from=builder /bin/load-tester ./load-tester
