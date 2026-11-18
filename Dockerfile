FROM golang:1.27-alpine AS builder

WORKDIR /src
RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/displaylab-socket ./cmd/server

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata \
  && adduser -D -H -u 10001 appuser

WORKDIR /app
COPY --from=builder /out/displaylab-socket /app/displaylab-socket

USER appuser
EXPOSE 8090

ENV HTTP_ADDR=:8090
ENV INTERNAL_API_KEY=displaylab-local-socket-key
ENV WS_ALLOW_ORIGINS=*

ENTRYPOINT ["/app/displaylab-socket"]
