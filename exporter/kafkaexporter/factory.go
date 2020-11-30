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

package kafkaexporter

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

const (
	typeStr             = "kafka"
	defaultTracesTopic  = "otlp_spans"
	defaultMetricsTopic = "otlp_metrics"
	defaultEncoding     = "otlp_proto"
	defaultBroker       = "localhost:9092"
	// default from sarama.NewConfig()
	defaultMetadataRetryMax = 3
	// default from sarama.NewConfig()
	defaultMetadataRetryBackoff = time.Millisecond * 250
	// default from sarama.NewConfig()
	defaultMetadataFull = true
)

// FactoryOption applies changes to kafkaExporterFactory.
type FactoryOption func(factory *kafkaExporterFactory)

// WithAddTracesMarshallers adds tracesMarshallers.
func WithAddTracesMarshallers(encodingMarshaller map[string]TracesMarshaller) FactoryOption {
	return func(factory *kafkaExporterFactory) {
		for encoding, marshaller := range encodingMarshaller {
			factory.tracesMarshallers[encoding] = marshaller
		}
	}
}

// NewFactory creates Kafka exporter factory.
func NewFactory(options ...FactoryOption) component.ExporterFactory {
	f := &kafkaExporterFactory{
		tracesMarshallers:  tracesMarshallers(),
		metricsMarshallers: metricsMarshallers(),
	}
	for _, o := range options {
		o(f)
	}
	return exporterhelper.NewFactory(
		typeStr,
		createDefaultConfig,
		exporterhelper.WithTraces(f.createTraceExporter),
		exporterhelper.WithMetrics(f.createMetricsExporter))
}

func createDefaultConfig() configmodels.Exporter {
	return &Config{
		ExporterSettings: configmodels.ExporterSettings{
			TypeVal: typeStr,
			NameVal: typeStr,
		},
		TimeoutSettings: exporterhelper.DefaultTimeoutSettings(),
		RetrySettings:   exporterhelper.DefaultRetrySettings(),
		QueueSettings:   exporterhelper.DefaultQueueSettings(),
		Brokers:         []string{defaultBroker},
		// using an empty topic to track when it has not been set by user, default is based on traces or metrics.
		Topic:    "",
		Encoding: defaultEncoding,
		Metadata: Metadata{
			Full: defaultMetadataFull,
			Retry: MetadataRetry{
				Max:     defaultMetadataRetryMax,
				Backoff: defaultMetadataRetryBackoff,
			},
		},
	}
}

type kafkaExporterFactory struct {
	tracesMarshallers  map[string]TracesMarshaller
	metricsMarshallers map[string]MetricsMarshaller
}

func (f *kafkaExporterFactory) createTraceExporter(
	_ context.Context,
	params component.ExporterCreateParams,
	cfg configmodels.Exporter,
) (component.TracesExporter, error) {
	oCfg := cfg.(*Config)
	if oCfg.Topic == "" {
		oCfg.Topic = defaultTracesTopic
	}
	exp, err := newTracesExporter(*oCfg, params, f.tracesMarshallers)
	if err != nil {
		return nil, err
	}
	return exporterhelper.NewTraceExporter(
		cfg,
		params.Logger,
		exp.traceDataPusher,
		// Disable exporterhelper Timeout, because we cannot pass a Context to the Producer,
		// and will rely on the sarama Producer Timeout logic.
		exporterhelper.WithTimeout(exporterhelper.TimeoutSettings{Timeout: 0}),
		exporterhelper.WithRetry(oCfg.RetrySettings),
		exporterhelper.WithQueue(oCfg.QueueSettings),
		exporterhelper.WithShutdown(exp.Close))
}

func (f *kafkaExporterFactory) createMetricsExporter(
	_ context.Context,
	params component.ExporterCreateParams,
	cfg configmodels.Exporter,
) (component.MetricsExporter, error) {
	oCfg := cfg.(*Config)
	if oCfg.Topic == "" {
		oCfg.Topic = defaultMetricsTopic
	}
	exp, err := newMetricsExporter(*oCfg, params, f.metricsMarshallers)
	if err != nil {
		return nil, err
	}
	return exporterhelper.NewMetricsExporter(
		cfg,
		params.Logger,
		exp.metricsDataPusher,
		// Disable exporterhelper Timeout, because we cannot pass a Context to the Producer,
		// and will rely on the sarama Producer Timeout logic.
		exporterhelper.WithTimeout(exporterhelper.TimeoutSettings{Timeout: 0}),
		exporterhelper.WithRetry(oCfg.RetrySettings),
		exporterhelper.WithQueue(oCfg.QueueSettings),
		exporterhelper.WithShutdown(exp.Close))
}
