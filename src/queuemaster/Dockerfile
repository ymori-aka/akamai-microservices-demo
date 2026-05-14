# Multi-stage build for queuemaster.
FROM golang:1.23-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /queuemaster .

# Distroless runtime — no shell, smaller attack surface.
FROM gcr.io/distroless/base-debian12

COPY --from=builder /queuemaster /queuemaster

EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/queuemaster"]
