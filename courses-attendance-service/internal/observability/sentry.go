package observability

import (
	"os"
	"time"

	"github.com/getsentry/sentry-go"
)

func initSentry() error {
	dsn := os.Getenv("SENTRY_DSN")
	if dsn == "" {
		return nil
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		Environment:      os.Getenv("ENV"),
		TracesSampleRate: 1.0,
	})
	if err != nil {
		return err
	}

	return nil
}

func FlushSentry() {
	sentry.Flush(2 * time.Second)
}
