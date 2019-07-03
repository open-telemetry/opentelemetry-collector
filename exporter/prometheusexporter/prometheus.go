// Copyright 2019, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
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

	"github.com/open-telemetry/opentelemetry-service/consumer"
	"github.com/open-telemetry/opentelemetry-service/consumer/consumerdata"
	"github.com/spf13/viper"

	// TODO: once this repository has been transferred to the
	// official census-ecosystem location, update this import path.
	"github.com/orijtech/prometheus-go-metrics-exporter"

	prometheus_golang "github.com/prometheus/client_golang/prometheus"
)

type prometheusConfig struct {
	// Namespace if set, exports metrics under the provided value.
	Namespace string `mapstructure:"namespace"`

	// ConstLabels are values that are applied for every exported metric.
	ConstLabels prometheus_golang.Labels `mapstructure:"const_labels"`

	// The address on which the Prometheus scrape handler will be run on.
	Address string `mapstructure:"address"`
}

var errBlankPrometheusAddress = errors.New("expecting a non-blank address to run the Prometheus metrics handler")

// PrometheusExportersFromViper unmarshals the viper and returns consumer.MetricsConsumers
// targeting Prometheus according to the configuration settings.
// It allows HTTP clients to scrape it on endpoint path "/metrics".
func PrometheusExportersFromViper(v *viper.Viper) (tps []consumer.TraceConsumer, mps []consumer.MetricsConsumer, doneFns []func() error, err error) {
	var cfg struct {
		Prometheus *prometheusConfig `mapstructure:"prometheus"`
	}
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, nil, nil, err
	}
	if cfg.Prometheus == nil {
		return nil, nil, nil, nil
	}

	pcfg := cfg.Prometheus
	addr := strings.TrimSpace(pcfg.Address)
	if addr == "" {
		err = errBlankPrometheusAddress
		return
	}

	opts := prometheus.Options{
		Namespace:   pcfg.Namespace,
		ConstLabels: pcfg.ConstLabels,
	}
	pe, err := prometheus.New(opts)
	if err != nil {
		return nil, nil, nil, err
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, nil, nil, err
	}

	// The Prometheus metrics exporter has to run on the provided address
	// as a server that'll be scraped by Prometheus.
	mux := http.NewServeMux()
	mux.Handle("/metrics", pe)

	srv := &http.Server{Handler: mux}
	go func() {
		_ = srv.Serve(ln)
	}()

	doneFns = append(doneFns, ln.Close)
	pexp := &prometheusExporter{exporter: pe}
	mps = append(mps, pexp)

	return
}

type prometheusExporter struct {
	exporter *prometheus.Exporter
}

var _ consumer.MetricsConsumer = (*prometheusExporter)(nil)

func (pe *prometheusExporter) ConsumeMetricsData(ctx context.Context, md consumerdata.MetricsData) error {
	for _, metric := range md.Metrics {
		_ = pe.exporter.ExportMetric(ctx, md.Node, md.Resource, metric)
	}
	return nil
}
