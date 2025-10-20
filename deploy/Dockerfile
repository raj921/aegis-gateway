FROM golang:1.21-alpine AS builder

WORKDIR /build

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY pkg/ ./pkg/
COPY policies/ ./policies/

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o aegis-gateway ./cmd/aegis

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /build/aegis-gateway .
COPY --from=builder /build/policies ./policies

RUN mkdir -p /app/logs && \
    addgroup -g 1000 aegis && \
    adduser -D -u 1000 -G aegis aegis && \
    chown -R aegis:aegis /app

USER aegis

EXPOSE 8080 8081 8082

CMD ["./aegis-gateway"]
