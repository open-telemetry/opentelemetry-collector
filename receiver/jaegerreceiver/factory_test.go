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

package jaegerreceiver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/config/configcheck"
	"github.com/open-telemetry/opentelemetry-collector/config/configerror"
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/open-telemetry/opentelemetry-collector/receiver"
)

func TestCreateDefaultConfig(t *testing.T) {
	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	assert.NotNil(t, cfg, "failed to create default config")
	assert.NoError(t, configcheck.ValidateConfig(cfg))
}

func TestCreateReceiver(t *testing.T) {
	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	// have to enable at least one protocol for the jaeger receiver to be created
	cfg.(*Config).Protocols[protoGRPC] = defaultsForProtocol(protoGRPC)

	tReceiver, err := factory.CreateTraceReceiver(context.Background(), zap.NewNop(), cfg, nil)
	assert.NoError(t, err, "receiver creation failed")
	assert.NotNil(t, tReceiver, "receiver creation failed")

	mReceiver, err := factory.CreateMetricsReceiver(zap.NewNop(), cfg, nil)
	assert.Equal(t, err, configerror.ErrDataTypeIsNotSupported)
	assert.Nil(t, mReceiver)
}

// default ports retrieved from https://www.jaegertracing.io/docs/1.16/deployment/
func TestCreateDefaultGRPCEndpoint(t *testing.T) {
	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	rCfg := cfg.(*Config)

	rCfg.Protocols[protoGRPC] = defaultsForProtocol(protoGRPC)
	r, err := factory.CreateTraceReceiver(context.Background(), zap.NewNop(), cfg, nil)

	assert.NoError(t, err, "unexpected error creating receiver")
	assert.Equal(t, 14250, r.(*jReceiver).config.CollectorGRPCPort, "grpc port should be default")
}

func TestCreateInvalidHTTPEndpoint(t *testing.T) {
	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	rCfg := cfg.(*Config)

	rCfg.Protocols[protoThriftHTTP] = defaultsForProtocol(protoThriftHTTP)
	r, err := factory.CreateTraceReceiver(context.Background(), zap.NewNop(), cfg, nil)

	assert.NoError(t, err, "unexpected error creating receiver")
	assert.Equal(t, 14268, r.(*jReceiver).config.CollectorHTTPPort, "http port should be default")
}

func TestCreateInvalidTChannelEndpoint(t *testing.T) {
	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	rCfg := cfg.(*Config)

	rCfg.Protocols[protoThriftTChannel] = defaultsForProtocol(protoThriftTChannel)
	r, err := factory.CreateTraceReceiver(context.Background(), zap.NewNop(), cfg, nil)

	assert.NoError(t, err, "unexpected error creating receiver")
	assert.Equal(t, 14267, r.(*jReceiver).config.CollectorThriftPort, "thrift port should be default")
}

func TestCreateInvalidThriftBinaryEndpoint(t *testing.T) {
	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	rCfg := cfg.(*Config)

	rCfg.Protocols[protoThriftBinary] = defaultsForProtocol(protoThriftBinary)
	r, err := factory.CreateTraceReceiver(context.Background(), zap.NewNop(), cfg, nil)

	assert.NoError(t, err, "unexpected error creating receiver")
	assert.Equal(t, 6832, r.(*jReceiver).config.AgentBinaryThriftPort, "thrift port should be default")
}

func TestCreateInvalidThriftCompactEndpoint(t *testing.T) {
	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	rCfg := cfg.(*Config)

	rCfg.Protocols[protoThriftCompact] = defaultsForProtocol(protoThriftCompact)
	r, err := factory.CreateTraceReceiver(context.Background(), zap.NewNop(), cfg, nil)

	assert.NoError(t, err, "unexpected error creating receiver")
	assert.Equal(t, 6831, r.(*jReceiver).config.AgentCompactThriftPort, "thrift port should be default")
}

func TestCreateNoPort(t *testing.T) {
	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	rCfg := cfg.(*Config)

	rCfg.Protocols[protoThriftHTTP] = &receiver.SecureReceiverSettings{
		ReceiverSettings: configmodels.ReceiverSettings{
			Endpoint: "localhost:",
		},
	}
	_, err := factory.CreateTraceReceiver(context.Background(), zap.NewNop(), cfg, nil)
	assert.Error(t, err, "receiver creation with no port number must fail")
}

func TestCreateLargePort(t *testing.T) {
	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	rCfg := cfg.(*Config)

	rCfg.Protocols[protoThriftHTTP] = &receiver.SecureReceiverSettings{
		ReceiverSettings: configmodels.ReceiverSettings{
			Endpoint: "localhost:65536",
		},
	}
	_, err := factory.CreateTraceReceiver(context.Background(), zap.NewNop(), cfg, nil)
	assert.Error(t, err, "receiver creation with too large port number must fail")
}

func TestCreateNoProtocols(t *testing.T) {
	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	rCfg := cfg.(*Config)

	rCfg.Protocols = make(map[string]*receiver.SecureReceiverSettings)

	_, err := factory.CreateTraceReceiver(context.Background(), zap.NewNop(), cfg, nil)
	assert.Error(t, err, "receiver creation with no protocols must fail")
}
