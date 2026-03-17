FROM golang:1.26-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=1 go build -o tenantai .

FROM alpine:3.21

RUN apk add --no-cache ca-certificates
RUN mkdir -p /app/data/services

WORKDIR /app
COPY --from=builder /build/tenantai .
COPY --from=builder /build/config.yaml .

EXPOSE 8080
CMD ["./tenantai"]
