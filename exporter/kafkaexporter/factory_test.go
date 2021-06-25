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
	"testing"

	"github.com/Shopify/sarama"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configcheck"
	"go.opentelemetry.io/collector/model/pdata"
)

func TestCreateDefaultConfig(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	assert.NotNil(t, cfg, "failed to create default config")
	assert.NoError(t, configcheck.ValidateConfig(cfg))
	assert.Equal(t, []string{defaultBroker}, cfg.Brokers)
	assert.Equal(t, "", cfg.Topic)
}

func TestCreateTracesExporter(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.Brokers = []string{"invalid:9092"}
	cfg.ProtocolVersion = "2.0.0"
	// this disables contacting the broker so we can successfully create the exporter
	cfg.Metadata.Full = false
	f := kafkaExporterFactory{tracesMarshalers: tracesMarshalers()}
	r, err := f.createTracesExporter(context.Background(), componenttest.NewNopExporterCreateSettings(), cfg)
	require.NoError(t, err)
	assert.NotNil(t, r)
}

func TestCreateMetricsExport(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.Brokers = []string{"invalid:9092"}
	cfg.ProtocolVersion = "2.0.0"
	// this disables contacting the broker so we can successfully create the exporter
	cfg.Metadata.Full = false
	mf := kafkaExporterFactory{metricsMarshalers: metricsMarshalers()}
	mr, err := mf.createMetricsExporter(context.Background(), componenttest.NewNopExporterCreateSettings(), cfg)
	require.NoError(t, err)
	assert.NotNil(t, mr)
}

func TestCreateLogsExport(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.Brokers = []string{"invalid:9092"}
	cfg.ProtocolVersion = "2.0.0"
	// this disables contacting the broker so we can successfully create the exporter
	cfg.Metadata.Full = false
	mf := kafkaExporterFactory{logsMarshalers: logsMarshalers()}
	mr, err := mf.createLogsExporter(context.Background(), componenttest.NewNopExporterCreateSettings(), cfg)
	require.NoError(t, err)
	assert.NotNil(t, mr)
}

func TestCreateTracesExporter_err(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.Brokers = []string{"invalid:9092"}
	cfg.ProtocolVersion = "2.0.0"
	f := kafkaExporterFactory{tracesMarshalers: tracesMarshalers()}
	r, err := f.createTracesExporter(context.Background(), componenttest.NewNopExporterCreateSettings(), cfg)
	// no available broker
	require.Error(t, err)
	assert.Nil(t, r)
}

func TestCreateMetricsExporter_err(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.Brokers = []string{"invalid:9092"}
	cfg.ProtocolVersion = "2.0.0"
	mf := kafkaExporterFactory{metricsMarshalers: metricsMarshalers()}
	mr, err := mf.createMetricsExporter(context.Background(), componenttest.NewNopExporterCreateSettings(), cfg)
	require.Error(t, err)
	assert.Nil(t, mr)
}

func TestCreateLogsExporter_err(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.Brokers = []string{"invalid:9092"}
	cfg.ProtocolVersion = "2.0.0"
	mf := kafkaExporterFactory{logsMarshalers: logsMarshalers()}
	mr, err := mf.createLogsExporter(context.Background(), componenttest.NewNopExporterCreateSettings(), cfg)
	require.Error(t, err)
	assert.Nil(t, mr)
}

func TestWithMarshalers(t *testing.T) {
	cm := &customMarshaler{}
	f := NewFactory(WithTracesMarshalers(cm))
	cfg := createDefaultConfig().(*Config)
	// disable contacting broker
	cfg.Metadata.Full = false

	t.Run("custom_encoding", func(t *testing.T) {
		cfg.Encoding = cm.Encoding()
		exporter, err := f.CreateTracesExporter(context.Background(), componenttest.NewNopExporterCreateSettings(), cfg)
		require.NoError(t, err)
		require.NotNil(t, exporter)
	})
	t.Run("default_encoding", func(t *testing.T) {
		cfg.Encoding = defaultEncoding
		exporter, err := f.CreateTracesExporter(context.Background(), componenttest.NewNopExporterCreateSettings(), cfg)
		require.NoError(t, err)
		assert.NotNil(t, exporter)
	})
}

type customMarshaler struct {
}

var _ TracesMarshaler = (*customMarshaler)(nil)

func (c customMarshaler) Marshal(_ pdata.Traces, topic string) ([]*sarama.ProducerMessage, error) {
	panic("implement me")
}

func (c customMarshaler) Encoding() string {
	return "custom"
}
