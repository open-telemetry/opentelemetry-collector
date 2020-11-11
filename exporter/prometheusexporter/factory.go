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
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

var errBlankPrometheusAddress = errors.New("expecting a non-blank address to run the Prometheus metrics handler")

const (
	// The value of "type" key in configuration.
	typeStr = "prometheus"
)

func NewFactory() component.ExporterFactory {
	return exporterhelper.NewFactory(
		typeStr,
		createDefaultConfig,
		exporterhelper.WithMetrics(createMetricsExporter))
}

func createDefaultConfig() configmodels.Exporter {
	return &Config{
		ExporterSettings: configmodels.ExporterSettings{
			TypeVal: typeStr,
			NameVal: typeStr,
		},
		ConstLabels:      map[string]string{},
		SendTimestamps:   false,
		MetricExpiration: time.Minute * 120,
	}
}

func createMetricsExporter(
	_ context.Context,
	params component.ExporterCreateParams,
	cfg configmodels.Exporter,
) (component.MetricsExporter, error) {
	pcfg := cfg.(*Config)

	addr := strings.TrimSpace(pcfg.Endpoint)
	if addr == "" {
		return nil, errBlankPrometheusAddress
	}

	reg := prometheus.NewRegistry()
	ph := promhttp.HandlerFor(
		reg,
		promhttp.HandlerOpts{
			ErrorHandling: promhttp.ContinueOnError,
		},
	)

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	// The Prometheus metrics exporter has to run on the provided address
	// as a server that'll be scraped by Prometheus.
	mux := http.NewServeMux()
	mux.Handle("/metrics", ph)

	srv := &http.Server{Handler: mux}
	go func() {
		_ = srv.Serve(ln)
	}()

	pexp := &prometheusExporter{
		name:         cfg.Name(),
		shutdownFunc: ln.Close,
		collector: &collector{
			registry:          reg,
			config:            pcfg,
			registeredMetrics: make(map[string]*prometheus.Desc),
			metricsValues:     make(map[string]*metricValue),
			logger:            params.Logger,
		},
	}

	return pexp, nil
}
