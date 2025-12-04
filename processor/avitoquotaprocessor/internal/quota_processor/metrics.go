package quota_processor

import (
	"context"
	"time"

	"go.avito.ru/arch/opentelemetry-collector-avito/processor/avitoquotaprocessor/internal/metadata"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	quotaTypeRateLimit  = "ratelimit"
	quotaTypeStoreLimit = "storelimit"
)

func incQuotaUsage(ctx context.Context, telemetry *metadata.TelemetryBuilder, ruleID string, days int64, service, system, severity string) {
	telemetry.ProcessorAvitoquotaRulesUsageCount.Add(
		ctx,
		1,
		metric.WithAttributes(
			attribute.String("qp_rule_id", ruleID),
			attribute.String("qp_quota_type", quotaTypeRateLimit),
		),
	)
	telemetry.ProcessorAvitoquotaRulesUsageCount.Add(
		ctx,
		days,
		metric.WithAttributes(
			attribute.String("qp_rule_id", ruleID),
			attribute.String("qp_quota_type", quotaTypeStoreLimit),
		),
	)
	telemetry.ProcessorAvitoquotaServiceUsageCount.Add(
		ctx,
		1,
		metric.WithAttributes(
			attribute.String("qp_rule_id", ruleID),
			attribute.String("qp_quota_type", quotaTypeRateLimit),
			attribute.String("qp_service", service),
			attribute.String("qp_system", system),
			attribute.String("qp_severity", severity),
		),
	)
	telemetry.ProcessorAvitoquotaServiceUsageCount.Add(
		ctx,
		days,
		metric.WithAttributes(
			attribute.String("qp_rule_id", ruleID),
			attribute.String("qp_quota_type", quotaTypeStoreLimit),
			attribute.String("qp_service", service),
			attribute.String("qp_system", system),
			attribute.String("qp_severity", severity),
		),
	)
}

func incBatchStats(ctx context.Context, telemetry *metadata.TelemetryBuilder, size int64, duration time.Duration) {
	telemetry.ProcessorAvitoquotaBatchCount.Add(ctx, 1)
	telemetry.ProcessorAvitoquotaBatchSize.Add(ctx, size)
	telemetry.ProcessorAvitoquotaBatchDurationUs.Record(ctx, duration.Microseconds())
	telemetry.ProcessorAvitoquotaAvgEventDurationNs.Record(ctx, duration.Nanoseconds()/size)
}
