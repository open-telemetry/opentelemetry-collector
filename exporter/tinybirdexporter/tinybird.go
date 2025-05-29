// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package tinybirdexporter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"time"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

const (
	headerRetryAfter  = "Retry-After"
	contentTypeNDJSON = "application/x-ndjson"
)

type tinybirdExporter struct {
	config    *Config
	client    *http.Client
	logger    *zap.Logger
	settings  component.TelemetrySettings
	userAgent string
}

func newExporter(cfg component.Config, set exporter.Settings) (*tinybirdExporter, error) {
	oCfg := cfg.(*Config)

	if err := oCfg.Validate(); err != nil {
		return nil, err
	}

	userAgent := fmt.Sprintf("%s/%s (%s/%s)",
		set.BuildInfo.Description, set.BuildInfo.Version, runtime.GOOS, runtime.GOARCH)

	return &tinybirdExporter{
		config:    oCfg,
		logger:    set.Logger,
		userAgent: userAgent,
		settings:  set.TelemetrySettings,
	}, nil
}

func (e *tinybirdExporter) start(ctx context.Context, host component.Host) error {
	e.client = &http.Client{
		Timeout: 30 * time.Second,
	}
	return nil
}

func (e *tinybirdExporter) pushTraces(ctx context.Context, td ptrace.Traces) error {
	events := make([]map[string]interface{}, 0, td.SpanCount())
	for i := 0; i < td.ResourceSpans().Len(); i++ {
		rs := td.ResourceSpans().At(i)
		for j := 0; j < rs.ScopeSpans().Len(); j++ {
			ss := rs.ScopeSpans().At(j)
			for k := 0; k < ss.Spans().Len(); k++ {
				span := ss.Spans().At(k)
				event := map[string]interface{}{
					"trace_id":       span.TraceID().String(),
					"span_id":        span.SpanID().String(),
					"parent_span_id": span.ParentSpanID().String(),
					"name":           span.Name(),
					"kind":           span.Kind().String(),
					"start_time":     span.StartTimestamp().AsTime().Format(time.RFC3339Nano),
					"end_time":       span.EndTimestamp().AsTime().Format(time.RFC3339Nano),
					"status_code":    span.Status().Code().String(),
					"status_message": span.Status().Message(),
					"attributes":     span.Attributes().AsRaw(),
					"events":         span.Events().Len(),
					"links":          span.Links().Len(),
				}
				events = append(events, event)
			}
		}
	}

	return e.export(ctx, "traces", events)
}

func (e *tinybirdExporter) pushMetrics(ctx context.Context, md pmetric.Metrics) error {
	events := make([]map[string]interface{}, 0, md.MetricCount())
	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		rm := md.ResourceMetrics().At(i)
		for j := 0; j < rm.ScopeMetrics().Len(); j++ {
			sm := rm.ScopeMetrics().At(j)
			for k := 0; k < sm.Metrics().Len(); k++ {
				metric := sm.Metrics().At(k)
				event := map[string]interface{}{
					"name":        metric.Name(),
					"description": metric.Description(),
					"unit":        metric.Unit(),
					"type":        metric.Type().String(),
				}

				switch metric.Type() {
				case pmetric.MetricTypeGauge:
					dps := metric.Gauge().DataPoints()
					for l := 0; l < dps.Len(); l++ {
						dp := dps.At(l)
						event["value"] = dp.DoubleValue()
						event["timestamp"] = dp.Timestamp().AsTime().Format(time.RFC3339Nano)
						event["attributes"] = dp.Attributes().AsRaw()
						events = append(events, event)
					}
				case pmetric.MetricTypeSum:
					dps := metric.Sum().DataPoints()
					for l := 0; l < dps.Len(); l++ {
						dp := dps.At(l)
						event["value"] = dp.DoubleValue()
						event["timestamp"] = dp.Timestamp().AsTime().Format(time.RFC3339Nano)
						event["attributes"] = dp.Attributes().AsRaw()
						events = append(events, event)
					}
				case pmetric.MetricTypeHistogram:
					dps := metric.Histogram().DataPoints()
					for l := 0; l < dps.Len(); l++ {
						dp := dps.At(l)
						event["count"] = dp.Count()
						event["sum"] = dp.Sum()
						event["timestamp"] = dp.Timestamp().AsTime().Format(time.RFC3339Nano)
						event["attributes"] = dp.Attributes().AsRaw()
						events = append(events, event)
					}
				}
			}
		}
	}

	return e.export(ctx, "metrics", events)
}

func (e *tinybirdExporter) pushLogs(ctx context.Context, ld plog.Logs) error {
	events := make([]map[string]interface{}, 0, ld.LogRecordCount())
	for i := 0; i < ld.ResourceLogs().Len(); i++ {
		rl := ld.ResourceLogs().At(i)
		for j := 0; j < rl.ScopeLogs().Len(); j++ {
			sl := rl.ScopeLogs().At(j)
			for k := 0; k < sl.LogRecords().Len(); k++ {
				log := sl.LogRecords().At(k)
				event := map[string]interface{}{
					"timestamp":  log.Timestamp().AsTime().Format(time.RFC3339Nano),
					"severity":   log.SeverityText(),
					"body":       log.Body().AsString(),
					"attributes": log.Attributes().AsRaw(),
					"trace_id":   log.TraceID().String(),
					"span_id":    log.SpanID().String(),
				}
				events = append(events, event)
			}
		}
	}

	return e.export(ctx, "logs", events)
}

func (e *tinybirdExporter) export(ctx context.Context, dataType string, events []map[string]interface{}) error {
	if len(events) == 0 {
		return nil
	}

	// Convert events to NDJSON
	var buf bytes.Buffer
	for _, event := range events {
		event["data_source"] = e.config.DataSource
		event["type"] = dataType
		event["timestamp"] = time.Now().Format(time.RFC3339Nano)

		jsonData, err := json.Marshal(event)
		if err != nil {
			return consumererror.NewPermanent(err)
		}
		buf.Write(jsonData)
		buf.WriteByte('\n')
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.config.Endpoint+"/v0/events?name="+dataType, &buf)
	if err != nil {
		return consumererror.NewPermanent(err)
	}

	// Set headers
	req.Header.Set("Content-Type", contentTypeNDJSON)
	req.Header.Set("Authorization", "Bearer "+e.config.Token)
	req.Header.Set("User-Agent", e.userAgent)

	// Send request
	resp, err := e.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Handle response
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	// Read error response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Check if retryable
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable {
		if retryAfter := resp.Header.Get(headerRetryAfter); retryAfter != "" {
			return exporterhelper.NewThrottleRetry(fmt.Errorf("request throttled, retry after %s: %s", retryAfter, string(body)), 0)
		}
		return exporterhelper.NewThrottleRetry(fmt.Errorf("request throttled: %s", string(body)), 0)
	}

	return consumererror.NewPermanent(fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body)))
}
