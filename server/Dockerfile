FROM golang:alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/mini-game .

FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends stockfish curl ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && useradd --system --uid 10001 --create-home app

WORKDIR /app

ENV STOCKFISH_PATH=/usr/games/stockfish

COPY --from=builder /out/mini-game /app/mini-game

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/healthz || exit 1

USER app

ENTRYPOINT ["/app/mini-game"]
