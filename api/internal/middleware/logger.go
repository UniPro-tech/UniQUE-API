package middleware

import (
	"context"
	"log/slog"
	"os"

	"github.com/UniPro-tech/UniQUE-API/api/internal/config"
	"go.opentelemetry.io/otel/trace"
)

// Log level 定義
var (
	SeverityDefault = slog.LevelInfo
	SeverityInfo    = slog.LevelInfo
	SeverityWarn    = slog.LevelWarn
	SeverityError   = slog.LevelError
	SeverityNotice  = slog.LevelInfo
)

// traceId , spanId 追加
type traceHandler struct {
	slog.Handler
}

// traceHandler 実装
func (h *traceHandler) Enabled(ctx context.Context, l slog.Level) bool {
	return h.Handler.Enabled(ctx, l)
}

func (h *traceHandler) Handle(ctx context.Context, r slog.Record) error {
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		r.AddAttrs(
			slog.String("trace_id", sc.TraceID().String()),
			slog.String("span_id", sc.SpanID().String()),
		)
	}
	return h.Handler.Handle(ctx, r)
}

func (h *traceHandler) WithAttr(attrs []slog.Attr) slog.Handler {
	return &traceHandler{h.Handler.WithAttrs(attrs)}
}

func (h *traceHandler) WithGroup(g string) slog.Handler {
	return h.Handler.WithGroup(g)
}

// logger 生成関数
func New() *slog.Logger {
	replacer := func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.MessageKey {
			a.Key = "message"
		}
		if a.Key == slog.LevelKey {
			a.Key = "level"
		}
		if a.Key == slog.SourceKey {
			a.Key = "source"
		}
		return a
	}
	cfg, _ := config.New()
	h := traceHandler{
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{AddSource: true, ReplaceAttr: replacer}),
	}
	newh := h.WithAttr([]slog.Attr{
		slog.Group("labels",
			slog.String("app", "MH-API"),
			slog.String("env", cfg.Env),
		),
	})
	logger := slog.New(newh)
	slog.SetDefault(logger)
	return logger
}
