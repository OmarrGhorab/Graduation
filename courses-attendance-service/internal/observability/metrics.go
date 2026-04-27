package observability

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests",
	}, []string{"method", "path", "status"})

	httpRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "Duration of HTTP requests in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path", "status"})
)

func MetricsMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Copy strings to prevent fasthttp byte slice mutation which corrupts Prometheus maps
		path := string(append([]byte(nil), c.Path()...))
		method := string(append([]byte(nil), c.Method()...))

		timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
		status := fmt.Sprintf("%d", c.Response().StatusCode())
			httpRequestDuration.WithLabelValues(method, path, status).Observe(v)
			httpRequestsTotal.WithLabelValues(method, path, status).Inc()
		}))
		defer timer.ObserveDuration()

		return c.Next()
	}
}

func SetupMetricsEndpoint(app *fiber.App) {
	app.Get("/metrics", func(c *fiber.Ctx) error {
		handler := promhttp.Handler()
		fasthttpadaptor.NewFastHTTPHandler(handler)(c.Context())
		return nil
	})
}
