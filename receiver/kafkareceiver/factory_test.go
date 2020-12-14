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

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configcheck"
	"go.opentelemetry.io/collector/consumer/pdata"
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

func TestCreateTraceReceiver(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.Brokers = []string{"invalid:9092"}
	cfg.ProtocolVersion = "2.0.0"
	f := kafkaReceiverFactory{unmarshalers: defaultUnmarshallers()}
	r, err := f.createTraceReceiver(context.Background(), component.ReceiverCreateParams{}, cfg, nil)
	// no available broker
	require.Error(t, err)
	assert.Nil(t, r)
}

func TestCreateTraceReceiver_error(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.ProtocolVersion = "2.0.0"
	// disable contacting broker at startup
	cfg.Metadata.Full = false
	f := kafkaReceiverFactory{unmarshalers: defaultUnmarshallers()}
	r, err := f.createTraceReceiver(context.Background(), component.ReceiverCreateParams{}, cfg, nil)
	require.NoError(t, err)
	assert.NotNil(t, r)
}

func TestWithUnmarshallers(t *testing.T) {
	unmarshaller := &customUnamarshaller{}
	f := NewFactory(WithAddUnmarshallers(map[string]Unmarshaller{unmarshaller.Encoding(): unmarshaller}))
	cfg := createDefaultConfig().(*Config)
	// disable contacting broker
	cfg.Metadata.Full = false
	cfg.ProtocolVersion = "2.0.0"

	t.Run("custom_encoding", func(t *testing.T) {
		cfg.Encoding = unmarshaller.Encoding()
		exporter, err := f.CreateTracesReceiver(context.Background(), component.ReceiverCreateParams{}, cfg, nil)
		require.NoError(t, err)
		require.NotNil(t, exporter)
	})
	t.Run("default_encoding", func(t *testing.T) {
		cfg.Encoding = new(otlpProtoUnmarshaller).Encoding()
		exporter, err := f.CreateTracesReceiver(context.Background(), component.ReceiverCreateParams{}, cfg, nil)
		require.NoError(t, err)
		assert.NotNil(t, exporter)
	})
}

type customUnamarshaller struct {
}

var _ Unmarshaller = (*customUnamarshaller)(nil)

func (c customUnamarshaller) Unmarshal(bytes []byte) (pdata.Traces, error) {
	panic("implement me")
}

func (c customUnamarshaller) Encoding() string {
	return "custom"
}
