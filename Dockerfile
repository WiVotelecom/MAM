# syntax=docker/dockerfile:1

# ---- Stage 1: build the frontend ----
FROM node:20-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm install
COPY frontend/ ./
RUN npm run build

# ---- Stage 2: build the Go binary (embeds the frontend) ----
FROM golang:1.25-alpine AS backend
WORKDIR /src
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Bring in the freshly built frontend so go:embed picks it up.
COPY --from=frontend /app/web/dist ./web/dist
ARG VERSION=0.1.0
RUN CGO_ENABLED=0 go build \
    -ldflags "-s -w -X github.com/WiVotelecom/MAM/internal/server.Version=${VERSION}" \
    -o /out/netinsight ./cmd/netinsight

# ---- Stage 3: minimal runtime ----
FROM alpine:3.20
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 netinsight \
    && mkdir -p /data && chown netinsight:netinsight /data
WORKDIR /data
COPY --from=backend /out/netinsight /usr/local/bin/netinsight
USER netinsight
EXPOSE 8080
ENV NETINSIGHT_ADDR=":8080" \
    NETINSIGHT_DB="/data/netinsight.db"
VOLUME ["/data"]
ENTRYPOINT ["/usr/local/bin/netinsight"]
