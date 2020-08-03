// Copyright 2020 The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cortexexporter

import (
	"bytes"
	"context"
	"net/http"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/prompb"
	"github.com/shurcooL/go/ctxhttp"

	"go.opentelemetry.io/collector/component"
	confighttp "go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

// This will be added to cortex_test, but currently I'm going to put it here in order to not have merge conflicts. Also, will readjust to fit our pipeline, not prometheus
type WriteRequest struct {
	Timeseries           []TimeSeries `protobuf:"bytes,1,rep,name=timeseries,proto3" json:"timeseries"`
	XXX_NoUnkeyedLiteral struct{}     `json:"-"`
	XXX_unrecognized     []byte       `json:"-"`
	XXX_sizecache        int32        `json:"-"`
}

func (c *Exporter) WrapTimeSeries(ts *prompb.TimeSeries) {
	return //will populate later
}

func (c *Exporter) Export(ctx context.Context, req *prompb.WriteRequest) error {
	//TODO:: Error handling
	data, err := proto.Marshal(req)
	if err != nil {
		return err
	}
	compressed := snappy.Encode(nil, data)
	httpReq, err := http.NewRequest("POST", c.url.String(), bytes.NewReader(compressed))
	if err != nil {
		return err
	}
	httpReq.Header.Add("Content-Encoding", "snappy")
	httpReq.Header.Set("Content-Type", "application/x-protobuf")
	httpReq.Header.Set("X-Prometheus-Remote-Write-Version", "0.1.0")
	httpReq = httpReq.WithContext(ctx)

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	httpResp, err := ctxhttp.Do(ctx, c.client, httpReq)
	return err
}

const (
	// The value of "type" key in configuration.
	typeStr = "cortex"
)

func NewFactory() component.ExporterFactory {
	return exporterhelper.NewFactory(
		typeStr,
		createDefaultConfig,
		exporterhelper.WithMetrics(createMetricsExporter))
}

func createMetricsExporter(_ context.Context, _ component.ExporterCreateParams, cfg configmodels.Exporter) (component.MetricsExporter, error) {

	cCfg := cfg.(*Config)
	client, err := cCfg.HTTPClientSettings.ToClient()
	if err != nil {
		return nil, err
	}
	ce := newCortexExporter(cCfg.Namespace, cCfg.HTTPClientSettings.Endpoint, client)
	cexp, err := exporterhelper.NewMetricsExporter(
		cfg,
		ce.pushMetrics,
		exporterhelper.WithTimeout(cCfg.TimeoutSettings),
		exporterhelper.WithQueue(cCfg.QueueSettings),
		exporterhelper.WithRetry(cCfg.RetrySettings),
		exporterhelper.WithShutdown(ce.shutdown),
	)
	if err != nil {
		return nil, err
	}

	return cexp, nil
}

func createDefaultConfig() configmodels.Exporter {
	// TODO: Enable the queued settings.
	qs := exporterhelper.CreateDefaultQueueSettings()
	qs.Enabled = false
	return &Config{
		ExporterSettings: configmodels.ExporterSettings{
			TypeVal: typeStr,
			NameVal: typeStr,
		},
		TimeoutSettings: exporterhelper.CreateDefaultTimeoutSettings(),
		RetrySettings:   exporterhelper.CreateDefaultRetrySettings(),
		QueueSettings:   qs,
		HTTPClientSettings: confighttp.HTTPClientSettings{
			// We almost read 0 bytes, so no need to tune ReadBufferSize.
			WriteBufferSize: 512 * 1024,
		},
	}
}

func createMetricsExporter(
	_ context.Context,
	_ component.ExporterCreateParams,
	cfg configmodels.Exporter,
) (component.MetricsExporter, error) {
	cCfg := cfg.(*Config)
	client,error := cCfg.HTTPClientSettings.ToClient()

	if error != nil {
		return nil, error
	}

	ce := newCortexExporter(cCfg.Namespace, cCfg.HTTPClientSettings.Endpoint,client)
	cexp, err := exporterhelper.NewMetricsExporter(
		cfg,
		ce.pushMetrics,
		exporterhelper.WithTimeout(cCfg.TimeoutSettings),
		exporterhelper.WithRetry(cCfg.RetrySettings),
		exporterhelper.WithQueue(cCfg.QueueSettings),
		exporterhelper.WithShutdown(ce.shutdown),
	)
	if err != nil {
		return nil, err
	}

	return cexp, nil
}
