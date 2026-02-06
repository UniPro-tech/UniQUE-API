FROM golang:1.20-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy rest of the sources
COPY . .

# Optional: install swag to generate docs if present (non-fatal)
RUN go install github.com/swaggo/swag/cmd/swag@latest || true
RUN swag init -g cmd/server/main.go || true

# Build arguments for embedding version metadata
ARG VERSION=dev
ARG COMMIT=none
ARG BRANCH=none

ENV CGO_ENABLED=0
RUN go build -ldflags "-s -w -X unibot/internal/config.Version=${VERSION} -X unibot/internal/config.GitCommit=${COMMIT} -X unibot/internal/config.GitBranch=${BRANCH}" -o /usr/local/bin/server cmd/server/main.go

FROM alpine:latest

RUN apk add --no-cache ca-certificates
WORKDIR /root/

COPY --from=builder /usr/local/bin/server .

ENV GIN_MODE=release
EXPOSE 8080
CMD ["./server"]