package observability

import (
	"context"
	"os"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.Logger

func initLogger() {
	config := zap.NewProductionConfig()
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	serviceName := os.Getenv("OTEL_SERVICE_NAME")
	if serviceName == "" {
		serviceName = "ws-gateway"
	}

	var err error
	Logger, err = config.Build(zap.Fields(
		zap.String("service_name", serviceName),
	))
	if err != nil {
		panic(err)
	}
}

func GetLoggerWithTrace(ctx context.Context) *zap.Logger {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return Logger.With(
			zap.String("trace_id", span.SpanContext().TraceID().String()),
			zap.String("span_id", span.SpanContext().SpanID().String()),
		)
	}
	return Logger
}
