// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlpreceiver

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

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
		require.NoError(f, err)
		if pb {
			req.Header.Add("Content-Type", pbContentType)
		} else {
			req.Header.Add("Content-Type", jsonContentType)
		}
		set := receivertest.NewNopSettings()
		set.TelemetrySettings = componenttest.NewNopTelemetrySettings()
		set.ID = otlpReceiverID
		resp := httptest.NewRecorder()
		switch handler % 3 {
		case 0:
			httpTracesReceiver, err := trace.New(consumertest.NewNop(), set, transportHTTP)
			require.NoError(f, err)
			handleTraces(resp, req, httpTracesReceiver)
		case 1:
			httpMetricsReceiver, err := metrics.New(consumertest.NewNop(), set, transportHTTP)
			require.NoError(f, err)
			handleMetrics(resp, req, httpMetricsReceiver)
		case 2:
			httpLogsReceiver, err := logs.New(consumertest.NewNop(), set, transportHTTP)
			require.NoError(f, err)
			handleLogs(resp, req, httpLogsReceiver)
		}
	})
}
