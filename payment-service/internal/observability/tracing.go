package observability

import (
	"context"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

func initTracing() (*sdktrace.TracerProvider, error) {
	ctx := context.Background()

	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "otel-collector:4318"
	}

	// Strip http:// or https:// if present for WithEndpoint
	cleanedEndpoint := endpoint
	if len(endpoint) > 7 && endpoint[:7] == "http://" {
		cleanedEndpoint = endpoint[7:]
	} else if len(endpoint) > 8 && endpoint[:8] == "https://" {
		cleanedEndpoint = endpoint[8:]
	}

	exporter, error := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(cleanedEndpoint),
		otlptracehttp.WithInsecure(),
	)
	if error != nil {
		return nil, error
	}

	serviceName := os.Getenv("OTEL_SERVICE_NAME")
	if serviceName == "" {
		serviceName = "payment-service"
	}

	res, error := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceVersionKey.String("1.0.0"),
		),
	)
	if error != nil {
		return nil, error
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return tp, nil
}
