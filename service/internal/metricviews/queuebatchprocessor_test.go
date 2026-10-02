// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package metricviews

import (
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config/configtelemetry"
)

func TestDefaultViewsQueueBatchProcessorMetrics(t *testing.T) {
	const scope = "go.opentelemetry.io/collector/processor/queuebatchprocessor"
	dropped := []string{
		"otelcol_processor_queuebatch_batch_send_size",
		"otelcol_processor_queuebatch_batch_send_size_bytes",
		"otelcol_processor_queuebatch_enqueue_size",
		"otelcol_processor_queuebatch_enqueue_size_bytes",
	}

	for _, level := range []configtelemetry.Level{
		configtelemetry.LevelBasic,
		configtelemetry.LevelNormal,
		configtelemetry.LevelDetailed,
	} {
		views := map[string]bool{}
		var filteredFailureAttrs []string
		for _, view := range DefaultViews(level) {
			if view.Selector == nil || view.Selector.MeterName == nil || *view.Selector.MeterName != scope ||
				view.Selector.InstrumentName == nil {
				continue
			}
			name := *view.Selector.InstrumentName
			views[name] = view.Stream != nil && view.Stream.Aggregation != nil
			if name == "otelcol_processor_queuebatch_send_failed_*" && view.Stream != nil &&
				view.Stream.AttributeKeys != nil {
				filteredFailureAttrs = view.Stream.AttributeKeys.Excluded
			}
		}

		if level == configtelemetry.LevelDetailed {
			require.Empty(t, views)
			continue
		}
		for _, name := range dropped {
			require.True(t, views[name], "level %s must drop %s", level, name)
		}
		require.Equal(t,
			[]string{"error.type", "error.permanent"},
			filteredFailureAttrs,
			"level %s must filter failure attributes",
		)
	}
}
