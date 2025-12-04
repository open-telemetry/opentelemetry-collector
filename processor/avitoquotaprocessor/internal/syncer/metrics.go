package syncer

import (
	"context"

	"go.avito.ru/arch/opentelemetry-collector-avito/processor/avitoquotaprocessor/internal/metadata"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func incSyncRules(ctx context.Context, telemetry *metadata.TelemetryBuilder, updateResult string) {
	telemetry.ProcessorAvitoquotaRulesUpdateAttempt.Add(
		ctx,
		1,
		metric.WithAttributes(attribute.String("qp_rules_update_result", updateResult)),
	)
}
