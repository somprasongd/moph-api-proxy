# syntax=docker/dockerfile:1
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o moph-ic-proxy ./cmd/server

FROM alpine:3.19
WORKDIR /srv/app
RUN adduser -D proxy
COPY --from=builder /app/moph-ic-proxy ./moph-ic-proxy
COPY moph-ic-proxy.env.example ./moph-ic-proxy.env.example
USER proxy
EXPOSE 3000
ENTRYPOINT ["./moph-ic-proxy"]
