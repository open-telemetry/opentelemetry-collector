package limiter

import (
	"context"

	"go.avito.ru/arch/opentelemetry-collector-avito/processor/avitoquotaprocessor/internal/metadata"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	attemptResultOk          = "ok"
	attemptResultRateLimited = "ratelimited"
)

func incRuleAttempt(ctx context.Context, telemetry *metadata.TelemetryBuilder, ruleID, attemptResult, limiterName string, n int64) {
	telemetry.ProcessorAvitoquotaTryRule.Add(
		ctx,
		n,
		metric.WithAttributes(
			attribute.String("qp_rule_id", ruleID),
			attribute.String("qp_rules_update_result", attemptResult),
			attribute.String("qp_limiter_name", limiterName),
		),
	)
}
