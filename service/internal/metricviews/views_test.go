// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package metricviews

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config/configtelemetry"
)

func TestDefaultViews(t *testing.T) {
	for _, tt := range []struct {
		name  string
		level configtelemetry.Level

		wantViewsCount int
	}{
		{
			name:           "None",
			level:          configtelemetry.LevelNone,
			wantViewsCount: 25,
		},
		{
			name:           "Basic",
			level:          configtelemetry.LevelBasic,
			wantViewsCount: 25,
		},
		{
			name:           "Normal",
			level:          configtelemetry.LevelNormal,
			wantViewsCount: 22,
		},
		{
			name:           "Detailed",
			level:          configtelemetry.LevelDetailed,
			wantViewsCount: 0,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			views := DefaultViews(tt.level)
			assert.Len(t, views, tt.wantViewsCount)
		})
	}
}

func TestDefaultViewsFiltersSendFailedAttributes(t *testing.T) {
	tests := []struct {
		name                         string
		level                        configtelemetry.Level
		expectSendFailedFilteredView bool
	}{
		{
			name:                         "basic level filters send_failed attributes",
			level:                        configtelemetry.LevelBasic,
			expectSendFailedFilteredView: true,
		},
		{
			name:                         "normal level filters send_failed attributes",
			level:                        configtelemetry.LevelNormal,
			expectSendFailedFilteredView: true,
		},
		{
			name:                         "detailed level does not filter send_failed attributes",
			level:                        configtelemetry.LevelDetailed,
			expectSendFailedFilteredView: false,
		},
	}

	// Both scopes document their send-failure attributes as detailed-level only.
	instruments := []string{
		"otelcol_exporter_send_failed_*",
		"otelcol_processor_queuebatch_send_failed_*",
	}

	for _, tt := range tests {
		for _, instrument := range instruments {
			t.Run(tt.name+"/"+instrument, func(t *testing.T) {
				views := DefaultViews(tt.level)

				foundSendFailedView := false
				for _, view := range views {
					if view.Selector == nil ||
						view.Selector.InstrumentName == nil ||
						*view.Selector.InstrumentName != instrument {
						continue
					}
					foundSendFailedView = true
					require.NotNil(t, view.Stream, "send_failed view should have a stream")
					require.NotNil(t, view.Stream.AttributeKeys, "send_failed view should have attribute keys")
					require.Equal(t, []string{"error.type", "error.permanent"}, view.Stream.AttributeKeys.Excluded,
						"send_failed view should exclude 'error.type' and 'error.permanent' attributes")
					break
				}

				if tt.expectSendFailedFilteredView {
					assert.True(t, foundSendFailedView,
						"Expected to find send_failed attribute filtering view at level %s", tt.level)
				} else {
					assert.False(t, foundSendFailedView,
						"Did not expect to find send_failed attribute filtering view at level %s", tt.level)
				}
			})
		}
	}
}

func TestDefaultViews_BatchExporterMetrics(t *testing.T) {
	tests := []struct {
		name             string
		level            configtelemetry.Level
		shouldDropBucket bool
		shouldDropBytes  bool
	}{
		{
			name:             "basic level drops bytes",
			level:            configtelemetry.LevelBasic,
			shouldDropBucket: true,
			shouldDropBytes:  true,
		},
		{
			name:             "normal level drops bytes",
			level:            configtelemetry.LevelNormal,
			shouldDropBucket: true,
			shouldDropBytes:  true,
		},
		{
			name:             "detailed level does not drop bytes",
			level:            configtelemetry.LevelDetailed,
			shouldDropBucket: false,
			shouldDropBytes:  false,
		},
	}

	scopes := []struct {
		scope            string
		bucketMetricName string
		bytesMetricName  string
	}{
		{
			scope:            "go.opentelemetry.io/collector/exporter/exporterhelper",
			bucketMetricName: "otelcol_exporter_queue_batch_send_size",
			bytesMetricName:  "otelcol_exporter_queue_batch_send_size_bytes",
		},
		{
			scope:            "go.opentelemetry.io/collector/exporter/exporterhelper",
			bucketMetricName: "otelcol_exporter_enqueue_size",
			bytesMetricName:  "otelcol_exporter_enqueue_size_bytes",
		},
		{
			scope:            "go.opentelemetry.io/collector/processor/queuebatchprocessor",
			bucketMetricName: "otelcol_processor_queuebatch_batch_send_size",
			bytesMetricName:  "otelcol_processor_queuebatch_batch_send_size_bytes",
		},
		{
			scope:            "go.opentelemetry.io/collector/processor/queuebatchprocessor",
			bucketMetricName: "otelcol_processor_queuebatch_enqueue_size",
			bytesMetricName:  "otelcol_processor_queuebatch_enqueue_size_bytes",
		},
	}

	for _, tt := range tests {
		for _, sc := range scopes {
			t.Run(tt.name+"/"+sc.bucketMetricName, func(t *testing.T) {
				views := DefaultViews(tt.level)

				var foundBucketDrop, foundBytesDrop bool
				for _, view := range views {
					if view.Selector == nil ||
						view.Selector.MeterName == nil || *view.Selector.MeterName != sc.scope ||
						view.Selector.InstrumentName == nil {
						continue
					}
					switch *view.Selector.InstrumentName {
					case sc.bucketMetricName:
						foundBucketDrop = true
					case sc.bytesMetricName:
						foundBytesDrop = true
					default:
						continue
					}
					// Verify it's a drop view
					require.NotNil(t, view.Stream)
					require.NotNil(t, view.Stream.Aggregation)
					require.NotNil(t, view.Stream.Aggregation.Drop)
				}

				assert.Equal(t, tt.shouldDropBucket, foundBucketDrop,
					"bucket metric drop view should be %v for level %v", tt.shouldDropBucket, tt.level)
				assert.Equal(t, tt.shouldDropBytes, foundBytesDrop,
					"bytes metric drop view should be %v for level %v", tt.shouldDropBytes, tt.level)
			})
		}
	}
}
