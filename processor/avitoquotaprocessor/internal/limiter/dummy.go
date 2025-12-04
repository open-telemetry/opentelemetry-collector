package limiter

import (
	"context"

	"go.avito.ru/arch/opentelemetry-collector-avito/processor/avitoquotaprocessor/internal/metadata"
)

type dummyLimiter struct {
	limiterName string
	telemetry   *metadata.TelemetryBuilder
}

func NewDummyLimiter(telemetry *metadata.TelemetryBuilder, name string) *dummyLimiter {
	return &dummyLimiter{
		limiterName: name,
		telemetry:   telemetry,
	}
}

func (l *dummyLimiter) Allow(id string, n int64) bool {
	incRuleAttempt(context.Background(), l.telemetry, id, attemptResultOk, l.limiterName, n)
	return true
}

func (l *dummyLimiter) Close() error {
	return nil
}
