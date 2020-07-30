package cortexexporter

import (
	otlp "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/metrics/v1"
)

func validateMetrics(descriptor *otlp.MetricDescriptor) bool {return true}

