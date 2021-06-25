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

package kafkareceiver

import (
	"context"
	"testing"

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
	assert.Equal(t, defaultTopic, cfg.Topic)
	assert.Equal(t, defaultGroupID, cfg.GroupID)
	assert.Equal(t, defaultClientID, cfg.ClientID)
}

func TestCreateTracesReceiver(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.Brokers = []string{"invalid:9092"}
	cfg.ProtocolVersion = "2.0.0"
	f := kafkaReceiverFactory{tracesUnmarshalers: defaultTracesUnmarshalers()}
	r, err := f.createTracesReceiver(context.Background(), componenttest.NewNopReceiverCreateSettings(), cfg, nil)
	// no available broker
	require.Error(t, err)
	assert.Nil(t, r)
}

func TestCreateTracesReceiver_error(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.ProtocolVersion = "2.0.0"
	// disable contacting broker at startup
	cfg.Metadata.Full = false
	f := kafkaReceiverFactory{tracesUnmarshalers: defaultTracesUnmarshalers()}
	r, err := f.createTracesReceiver(context.Background(), componenttest.NewNopReceiverCreateSettings(), cfg, nil)
	require.NoError(t, err)
	assert.NotNil(t, r)
}

func TestWithTracesUnmarshalers(t *testing.T) {
	unmarshaler := &customTracesUnmarshaler{}
	f := NewFactory(WithTracesUnmarshalers(unmarshaler))
	cfg := createDefaultConfig().(*Config)
	// disable contacting broker
	cfg.Metadata.Full = false
	cfg.ProtocolVersion = "2.0.0"

	t.Run("custom_encoding", func(t *testing.T) {
		cfg.Encoding = unmarshaler.Encoding()
		receiver, err := f.CreateTracesReceiver(context.Background(), componenttest.NewNopReceiverCreateSettings(), cfg, nil)
		require.NoError(t, err)
		require.NotNil(t, receiver)
	})
	t.Run("default_encoding", func(t *testing.T) {
		cfg.Encoding = defaultEncoding
		receiver, err := f.CreateTracesReceiver(context.Background(), componenttest.NewNopReceiverCreateSettings(), cfg, nil)
		require.NoError(t, err)
		assert.NotNil(t, receiver)
	})
}

func TestCreateMetricsReceiver(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.Brokers = []string{"invalid:9092"}
	cfg.ProtocolVersion = "2.0.0"
	f := kafkaReceiverFactory{metricsUnmarshalers: defaultMetricsUnmarshalers()}
	r, err := f.createMetricsReceiver(context.Background(), componenttest.NewNopReceiverCreateSettings(), cfg, nil)
	// no available broker
	require.Error(t, err)
	assert.Nil(t, r)
}

func TestCreateMetricsReceiver_error(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.ProtocolVersion = "2.0.0"
	// disable contacting broker at startup
	cfg.Metadata.Full = false
	f := kafkaReceiverFactory{metricsUnmarshalers: defaultMetricsUnmarshalers()}
	r, err := f.createMetricsReceiver(context.Background(), componenttest.NewNopReceiverCreateSettings(), cfg, nil)
	require.NoError(t, err)
	assert.NotNil(t, r)
}

func TestWithMetricsUnmarshalers(t *testing.T) {
	unmarshaler := &customMetricsUnmarshaler{}
	f := NewFactory(WithMetricsUnmarshalers(unmarshaler))
	cfg := createDefaultConfig().(*Config)
	// disable contacting broker
	cfg.Metadata.Full = false
	cfg.ProtocolVersion = "2.0.0"

	t.Run("custom_encoding", func(t *testing.T) {
		cfg.Encoding = unmarshaler.Encoding()
		receiver, err := f.CreateMetricsReceiver(context.Background(), componenttest.NewNopReceiverCreateSettings(), cfg, nil)
		require.NoError(t, err)
		require.NotNil(t, receiver)
	})
	t.Run("default_encoding", func(t *testing.T) {
		cfg.Encoding = defaultEncoding
		receiver, err := f.CreateMetricsReceiver(context.Background(), componenttest.NewNopReceiverCreateSettings(), cfg, nil)
		require.NoError(t, err)
		assert.NotNil(t, receiver)
	})
}

func TestCreateLogsReceiver(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.Brokers = []string{"invalid:9092"}
	cfg.ProtocolVersion = "2.0.0"
	f := kafkaReceiverFactory{logsUnmarshalers: defaultLogsUnmarshalers()}
	r, err := f.createLogsReceiver(context.Background(), componenttest.NewNopReceiverCreateSettings(), cfg, nil)
	// no available broker
	require.Error(t, err)
	assert.Nil(t, r)
}

func TestCreateLogsReceiver_error(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.ProtocolVersion = "2.0.0"
	// disable contacting broker at startup
	cfg.Metadata.Full = false
	f := kafkaReceiverFactory{logsUnmarshalers: defaultLogsUnmarshalers()}
	r, err := f.createLogsReceiver(context.Background(), componenttest.NewNopReceiverCreateSettings(), cfg, nil)
	require.NoError(t, err)
	assert.NotNil(t, r)
}

func TestWithLogsUnmarshalers(t *testing.T) {
	unmarshaler := &customLogsUnmarshaler{}
	f := NewFactory(WithLogsUnmarshalers(unmarshaler))
	cfg := createDefaultConfig().(*Config)
	// disable contacting broker
	cfg.Metadata.Full = false
	cfg.ProtocolVersion = "2.0.0"

	t.Run("custom_encoding", func(t *testing.T) {
		cfg.Encoding = unmarshaler.Encoding()
		exporter, err := f.CreateLogsReceiver(context.Background(), componenttest.NewNopReceiverCreateSettings(), cfg, nil)
		require.NoError(t, err)
		require.NotNil(t, exporter)
	})
	t.Run("default_encoding", func(t *testing.T) {
		cfg.Encoding = defaultEncoding
		exporter, err := f.CreateLogsReceiver(context.Background(), componenttest.NewNopReceiverCreateSettings(), cfg, nil)
		require.NoError(t, err)
		assert.NotNil(t, exporter)
	})
}

type customTracesUnmarshaler struct {
}

type customMetricsUnmarshaler struct {
}

type customLogsUnmarshaler struct {
}

var _ TracesUnmarshaler = (*customTracesUnmarshaler)(nil)

func (c customTracesUnmarshaler) Unmarshal([]byte) (pdata.Traces, error) {
	panic("implement me")
}

func (c customTracesUnmarshaler) Encoding() string {
	return "custom"
}

func (c customMetricsUnmarshaler) Unmarshal([]byte) (pdata.Metrics, error) {
	panic("implement me")
}

func (c customMetricsUnmarshaler) Encoding() string {
	return "custom"
}

func (c customLogsUnmarshaler) Unmarshal([]byte) (pdata.Logs, error) {
	panic("implement me")
}

func (c customLogsUnmarshaler) Encoding() string {
	return "custom"
}
