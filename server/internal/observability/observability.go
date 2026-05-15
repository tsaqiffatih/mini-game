package observability

import (
	"bufio"
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"
)

type contextKey string

const (
	traceIDKey contextKey = "trace_id"
)

var logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
	Level: slog.LevelInfo,
}))

func Init(serviceName string) {
	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger.With("service", serviceName))
}

func Logger() *slog.Logger {
	return slog.Default()
}

func WithTraceID(ctx context.Context, traceID string) context.Context {
	if traceID == "" {
		return ctx
	}
	return context.WithValue(ctx, traceIDKey, traceID)
}

func TraceID(ctx context.Context) string {
	traceID, _ := ctx.Value(traceIDKey).(string)
	return traceID
}

func StartSpan(ctx context.Context, name string, attrs ...slog.Attr) (context.Context, func(error)) {
	startedAt := time.Now()
	fields := []slog.Attr{
		slog.String("event_type", "trace_span_start"),
		slog.String("span", name),
	}
	fields = append(fields, traceAttrs(ctx)...)
	fields = append(fields, attrs...)
	Logger().LogAttrs(ctx, slog.LevelDebug, "trace span started", fields...)

	return ctx, func(err error) {
		fields := []slog.Attr{
			slog.String("event_type", "trace_span_end"),
			slog.String("span", name),
			slog.Duration("duration", time.Since(startedAt)),
		}
		fields = append(fields, traceAttrs(ctx)...)
		fields = append(fields, attrs...)
		if err != nil {
			fields = append(fields, slog.String("error", err.Error()))
			Logger().LogAttrs(ctx, slog.LevelWarn, "trace span ended with error", fields...)
			return
		}
		Logger().LogAttrs(ctx, slog.LevelDebug, "trace span ended", fields...)
	}
}

func RoomAttrs(roomID, playerID, eventType string) []slog.Attr {
	return []slog.Attr{
		slog.String("room_id", roomID),
		slog.String("player_id", playerID),
		slog.String("event_type", eventType),
	}
}

func RequestMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		traceID := r.Header.Get("traceparent")
		if traceID == "" {
			traceID = r.Header.Get("X-Request-ID")
		}
		ctx := WithTraceID(r.Context(), traceID)

		recorder := &responseRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(recorder, r.WithContext(ctx))

		attrs := []slog.Attr{
			slog.String("event_type", "http_request"),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", recorder.statusCode),
			slog.Duration("duration", time.Since(startedAt)),
			slog.String("remote_addr", r.RemoteAddr),
		}
		attrs = append(attrs, traceAttrs(ctx)...)
		Logger().LogAttrs(ctx, slog.LevelInfo, "request completed", attrs...)
	})
}

func traceAttrs(ctx context.Context) []slog.Attr {
	traceID := TraceID(ctx)
	if traceID == "" {
		return nil
	}
	return []slog.Attr{slog.String("trace_id", traceID)}
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *responseRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}
	return hijacker.Hijack()
}

func (r *responseRecorder) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}
