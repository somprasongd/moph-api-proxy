# MOPH IC Proxy (Go)

This service provides an HTTP gateway to MOPH IC, FDH, and related APIs. It replaces the original Node.js implementation with a Go application focused on detailed operational logging and resilient token handling.

## Features

- Automatic token acquisition and refresh for MOPH IC and FDH services with Redis-backed caching
- Reverse proxy capable of forwarding GET, POST, PUT, PATCH, and DELETE requests to configured upstreams
- Optional API key enforcement using a generated secret stored on disk
- HTML interface for status, API key display, and credential rotation workflows
- Extensive logging that captures request lifecycle and error details

## Requirements

- Go 1.24 or newer (the code is forward compatible with Go 1.25)
- Redis 5 or newer
- Access credentials for the upstream MOPH services

## Configuration

Configuration is sourced from environment variables. The defaults mirror the previous Node.js project and are documented in `moph-api-proxy.env.example`.

| Variable | Description |
| --- | --- |
| `APP_PORT` | Port for the HTTP server (default `3000`) |
| `REDIS_HOST` / `REDIS_PORT` / `REDIS_PASSWORD` | Redis connection details |
| `MOPH_*`, `FDH_*`, `EPIDEM_API` | Upstream endpoints and secrets |
| `MOPH_HCODE` | Hospital code (required) |
| `USE_API_KEY` | Enables API key validation (default `true`) |

## Running Locally

```bash
# Install dependencies (none outside of the Go standard library)
go build ./cmd/server
./cmd/server
```

Ensure a Redis instance is running and accessible based on your environment variables before starting the proxy.

## Docker

A multi-stage Dockerfile is provided:

```bash
docker build -t moph-api-proxy .
docker run --env-file moph-api-proxy.env.example -p 3000:3000 moph-api-proxy
```

For development with Redis you can use the included `docker-compose.yml`.

## Logging

The server emits detailed logs for all requests. Error logs include contextual data to aid troubleshooting, especially around upstream request failures and token refresh events.
