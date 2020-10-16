// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package otlphttpexporter

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/pdata"
)

type exporterImp struct {
	// Input configuration.
	config     *Config
	client     *http.Client
	tracesURL  string
	metricsURL string
	logsURL    string
}

// Crete new exporter.
func newExporter(cfg configmodels.Exporter) (*exporterImp, error) {
	oCfg := cfg.(*Config)

	if oCfg.Endpoint != "" {
		_, err := url.Parse(oCfg.Endpoint)
		if err != nil {
			return nil, errors.New("endpoint must be a valid URL")
		}
	}

	client, err := oCfg.HTTPClientSettings.ToClient()
	if err != nil {
		return nil, err
	}

	return &exporterImp{
		config: oCfg,
		client: client,
	}, nil
}

func (e *exporterImp) pushTraceData(ctx context.Context, traces pdata.Traces) (int, error) {
	request, err := traces.ToOtlpProtoBytes()
	if err != nil {
		return traces.SpanCount(), consumererror.Permanent(err)
	}

	err = e.export(ctx, e.tracesURL, request)
	if err != nil {
		return traces.SpanCount(), err
	}

	return 0, nil
}

func (e *exporterImp) pushMetricsData(ctx context.Context, metrics pdata.Metrics) (int, error) {
	request, err := metrics.ToOtlpProtoBytes()
	if err != nil {
		return metrics.MetricCount(), consumererror.Permanent(err)
	}

	err = e.export(ctx, e.metricsURL, request)
	if err != nil {
		return metrics.MetricCount(), err
	}

	return 0, nil
}

func (e *exporterImp) pushLogData(ctx context.Context, logs pdata.Logs) (int, error) {
	request, err := logs.ToOtlpProtoBytes()
	if err != nil {
		return logs.LogRecordCount(), consumererror.Permanent(err)
	}

	err = e.export(ctx, e.logsURL, request)
	if err != nil {
		return logs.LogRecordCount(), err
	}

	return 0, nil
}

func (e *exporterImp) export(ctx context.Context, url string, request []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(request))
	if err != nil {
		return consumererror.Permanent(err)
	}
	req.Header.Set("Content-Type", "application/x-protobuf")

	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to push trace data via OTLP exporter: %w", err)
	}

	_ = resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		// TODO: Parse status and decide if can or not retry.
		// TODO: Parse status and decide throttling.
		return fmt.Errorf("failed the request with status code %d", resp.StatusCode)
	}

	return nil
}
