FROM golang:1.26-trixie AS builder

RUN apt-get update && apt-get install -y --no-install-recommends gcc g++ && rm -rf /var/lib/apt/lists/*

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=1 go build -o tenantai .

FROM debian:trixie-slim

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*
RUN mkdir -p /app/data/services

WORKDIR /app
COPY --from=builder /build/tenantai .
COPY --from=builder /build/config.example.yaml ./config.yaml

EXPOSE 8080
CMD ["./tenantai"]
