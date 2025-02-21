// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlpreceiver

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver/otlpreceiver/internal/logs"
	"go.opentelemetry.io/collector/receiver/otlpreceiver/internal/metrics"
	"go.opentelemetry.io/collector/receiver/otlpreceiver/internal/trace"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func FuzzReceiverHandlers(f *testing.F) {
	f.Fuzz(func(_ *testing.T, data []byte, pb bool, handler int) {
		req, err := http.NewRequest(http.MethodPost, "", bytes.NewReader(data))
		if err != nil {
			return
		}
		if pb {
			req.Header.Add("Content-Type", pbContentType)
		} else {
			req.Header.Add("Content-Type", jsonContentType)
		}
		set := receivertest.NewNopSettings(receivertest.NopType)
		set.TelemetrySettings = componenttest.NewNopTelemetrySettings()
		set.ID = otlpReceiverID
		cfg := createDefaultConfig().(*Config)
		r, err := newOtlpReceiver(cfg, &set)
		if err != nil {
			panic(err)
		}
		r.nextTraces = consumertest.NewNop()
		r.nextLogs = consumertest.NewNop()
		r.nextMetrics = consumertest.NewNop()
		r.nextProfiles = consumertest.NewNop()
		resp := httptest.NewRecorder()
		switch handler % 3 {
		case 0:
			httpTracesReceiver := trace.New(r.nextTraces, r.obsrepHTTP)
			handleTraces(resp, req, httpTracesReceiver)
		case 1:
			httpMetricsReceiver := metrics.New(r.nextMetrics, r.obsrepHTTP)
			handleMetrics(resp, req, httpMetricsReceiver)
		case 2:
			httpLogsReceiver := logs.New(r.nextLogs, r.obsrepHTTP)
			handleLogs(resp, req, httpLogsReceiver)
		}
	})
}
