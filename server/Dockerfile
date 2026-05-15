FROM golang:1.23-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/mini-game .

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=builder /out/mini-game /app/mini-game
COPY --from=builder /src/stockfish/stockfish /app/stockfish/stockfish

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/app/mini-game"]
