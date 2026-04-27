package observability

import (
	"context"
	"log"

	"github.com/gofiber/contrib/fibersentry"
	"github.com/gofiber/contrib/otelfiber/v2"
	"github.com/gofiber/fiber/v2"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type Observability struct {
	TracerProvider *sdktrace.TracerProvider
}

func Init(app *fiber.App) *Observability {
	initLogger()

	tp, err := initTracing()
	if err != nil {
		log.Printf("Failed to initialize tracing: %v", err)
	}

	err = initSentry()
	if err != nil {
		log.Printf("Failed to initialize Sentry: %v", err)
	}

	// OTel Fiber middleware — creates a span for every incoming HTTP request
	app.Use(otelfiber.Middleware())

	// Sentry Fiber middleware — captures panics and reports to Sentry
	app.Use(fibersentry.New(fibersentry.Config{
		Repanic: true,
	}))

	// Setup metrics endpoint
	SetupMetricsEndpoint(app)

	// Add metrics middleware
	app.Use(MetricsMiddleware())

	return &Observability{
		TracerProvider: tp,
	}
}

func (o *Observability) Shutdown() {
	if o.TracerProvider != nil {
		o.TracerProvider.Shutdown(context.Background())
	}
	FlushSentry()
}
