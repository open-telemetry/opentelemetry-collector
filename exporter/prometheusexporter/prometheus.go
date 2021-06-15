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

package prometheusexporter

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/obsreport"
)

type prometheusExporter struct {
	name         string
	endpoint     string
	shutdownFunc func() error
	handler      http.Handler
	collector    *collector
	registry     *prometheus.Registry
	obsrep       *obsreport.Exporter
}

var errBlankPrometheusAddress = errors.New("expecting a non-blank address to run the Prometheus metrics handler")

func newPrometheusExporter(config *Config, logger *zap.Logger) (*prometheusExporter, error) {
	addr := strings.TrimSpace(config.Endpoint)
	if strings.TrimSpace(config.Endpoint) == "" {
		return nil, errBlankPrometheusAddress
	}

	obsrep := obsreport.NewExporter(obsreport.ExporterSettings{
		Level:      configtelemetry.GetMetricsLevelFlagValue(),
		ExporterID: config.ID(),
	})

	collector := newCollector(config, logger)
	registry := prometheus.NewRegistry()
	_ = registry.Register(collector)

	return &prometheusExporter{
		name:         config.ID().String(),
		endpoint:     addr,
		collector:    collector,
		registry:     registry,
		shutdownFunc: func() error { return nil },
		obsrep:       obsrep,
		handler: promhttp.HandlerFor(
			registry,
			promhttp.HandlerOpts{
				ErrorHandling: promhttp.ContinueOnError,
			},
		),
	}, nil
}

func (pe *prometheusExporter) Start(_ context.Context, _ component.Host) error {
	ln, err := net.Listen("tcp", pe.endpoint)
	if err != nil {
		return err
	}

	pe.shutdownFunc = ln.Close

	mux := http.NewServeMux()
	mux.Handle("/metrics", pe.handler)
	srv := &http.Server{Handler: mux}
	go func() {
		_ = srv.Serve(ln)
	}()

	return nil
}

func (pe *prometheusExporter) ConsumeMetrics(ctx context.Context, md pdata.Metrics) error {
	pe.obsrep.StartMetricsOp(ctx)
	n := 0
	rmetrics := md.ResourceMetrics()
	for i := 0; i < rmetrics.Len(); i++ {
		n += pe.collector.processMetrics(rmetrics.At(i))
	}
	pe.obsrep.EndMetricsOp(ctx, n, nil)

	return nil
}

func (pe *prometheusExporter) Shutdown(context.Context) error {
	return pe.shutdownFunc()
}
