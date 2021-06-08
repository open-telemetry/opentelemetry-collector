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

package jaegerreceiver

import (
	"context"
	"fmt"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configcheck"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/config/configtest"
	"go.opentelemetry.io/collector/config/configtls"
)

func TestTypeStr(t *testing.T) {
	factory := NewFactory()

	assert.Equal(t, "jaeger", string(factory.Type()))
}

func TestCreateDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.NotNil(t, cfg, "failed to create default config")
	assert.NoError(t, configcheck.ValidateConfig(cfg))
}

func TestCreateReceiver(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	// have to enable at least one protocol for the jaeger receiver to be created
	cfg.(*Config).Protocols.GRPC = &configgrpc.GRPCServerSettings{
		NetAddr: confignet.NetAddr{
			Endpoint:  defaultGRPCBindEndpoint,
			Transport: "tcp",
		},
	}
	set := componenttest.NewNopReceiverCreateSettings()
	tReceiver, err := factory.CreateTracesReceiver(context.Background(), set, cfg, nil)
	assert.NoError(t, err, "receiver creation failed")
	assert.NotNil(t, tReceiver, "receiver creation failed")

	mReceiver, err := factory.CreateMetricsReceiver(context.Background(), set, cfg, nil)
	assert.Equal(t, err, componenterror.ErrDataTypeIsNotSupported)
	assert.Nil(t, mReceiver)
}

func TestCreateReceiverGeneralConfig(t *testing.T) {
	factories, err := componenttest.NopFactories()
	assert.NoError(t, err)

	factory := NewFactory()
	factories.Receivers[typeStr] = factory

	cfg, err := configtest.LoadConfigAndValidate(path.Join(".", "testdata", "config.yaml"), factories)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	rCfg, ok := cfg.Receivers[config.NewIDWithName(typeStr, "customname")]
	require.True(t, ok)

	set := componenttest.NewNopReceiverCreateSettings()
	tReceiver, err := factory.CreateTracesReceiver(context.Background(), set, rCfg, nil)
	assert.NoError(t, err, "receiver creation failed")
	assert.NotNil(t, tReceiver, "receiver creation failed")

	mReceiver, err := factory.CreateMetricsReceiver(context.Background(), set, rCfg, nil)
	assert.Equal(t, err, componenterror.ErrDataTypeIsNotSupported)
	assert.Nil(t, mReceiver)
}

// default ports retrieved from https://www.jaegertracing.io/docs/1.16/deployment/
func TestCreateDefaultGRPCEndpoint(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	cfg.(*Config).Protocols.GRPC = &configgrpc.GRPCServerSettings{
		NetAddr: confignet.NetAddr{
			Endpoint:  defaultGRPCBindEndpoint,
			Transport: "tcp",
		},
	}
	set := componenttest.NewNopReceiverCreateSettings()
	r, err := factory.CreateTracesReceiver(context.Background(), set, cfg, nil)

	assert.NoError(t, err, "unexpected error creating receiver")
	assert.Equal(t, 14250, r.(*jReceiver).config.CollectorGRPCPort, "grpc port should be default")
}

func TestCreateTLSGPRCEndpoint(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	cfg.(*Config).Protocols.GRPC = &configgrpc.GRPCServerSettings{
		NetAddr: confignet.NetAddr{
			Endpoint:  defaultGRPCBindEndpoint,
			Transport: "tcp",
		},
		TLSSetting: &configtls.TLSServerSetting{
			TLSSetting: configtls.TLSSetting{
				CertFile: "./testdata/server.crt",
				KeyFile:  "./testdata/server.key",
			},
		},
	}
	set := componenttest.NewNopReceiverCreateSettings()

	_, err := factory.CreateTracesReceiver(context.Background(), set, cfg, nil)
	assert.NoError(t, err, "tls-enabled receiver creation failed")
}

func TestCreateTLSThriftHTTPEndpoint(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	cfg.(*Config).Protocols.ThriftHTTP = &confighttp.HTTPServerSettings{
		Endpoint: defaultHTTPBindEndpoint,
		TLSSetting: &configtls.TLSServerSetting{
			TLSSetting: configtls.TLSSetting{
				CertFile: "./testdata/server.crt",
				KeyFile:  "./testdata/server.key",
			},
		},
	}

	set := componenttest.NewNopReceiverCreateSettings()

	_, err := factory.CreateTracesReceiver(context.Background(), set, cfg, nil)
	assert.NoError(t, err, "tls-enabled receiver creation failed")
}

func TestCreateInvalidHTTPEndpoint(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	cfg.(*Config).Protocols.ThriftHTTP = &confighttp.HTTPServerSettings{
		Endpoint: defaultHTTPBindEndpoint,
	}
	set := componenttest.NewNopReceiverCreateSettings()
	r, err := factory.CreateTracesReceiver(context.Background(), set, cfg, nil)

	assert.NoError(t, err, "unexpected error creating receiver")
	assert.Equal(t, 14268, r.(*jReceiver).config.CollectorHTTPPort, "http port should be default")
}

func TestCreateInvalidThriftBinaryEndpoint(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	cfg.(*Config).Protocols.ThriftBinary = &ProtocolUDP{
		Endpoint: defaultThriftBinaryBindEndpoint,
	}
	set := componenttest.NewNopReceiverCreateSettings()
	r, err := factory.CreateTracesReceiver(context.Background(), set, cfg, nil)

	assert.NoError(t, err, "unexpected error creating receiver")
	assert.Equal(t, 6832, r.(*jReceiver).config.AgentBinaryThriftPort, "thrift port should be default")
}

func TestCreateInvalidThriftCompactEndpoint(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	cfg.(*Config).Protocols.ThriftCompact = &ProtocolUDP{
		Endpoint: defaultThriftCompactBindEndpoint,
	}
	set := componenttest.NewNopReceiverCreateSettings()
	r, err := factory.CreateTracesReceiver(context.Background(), set, cfg, nil)

	assert.NoError(t, err, "unexpected error creating receiver")
	assert.Equal(t, 6831, r.(*jReceiver).config.AgentCompactThriftPort, "thrift port should be default")
}

func TestDefaultAgentRemoteSamplingEndpointAndPort(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	rCfg := cfg.(*Config)

	rCfg.Protocols.ThriftCompact = &ProtocolUDP{
		Endpoint: defaultThriftCompactBindEndpoint,
	}
	rCfg.RemoteSampling = &RemoteSamplingConfig{}
	set := componenttest.NewNopReceiverCreateSettings()
	r, err := factory.CreateTracesReceiver(context.Background(), set, cfg, nil)

	assert.NoError(t, err, "create trace receiver should not error")
	assert.Equal(t, defaultGRPCBindEndpoint, r.(*jReceiver).config.RemoteSamplingClientSettings.Endpoint)
	assert.Equal(t, defaultAgentRemoteSamplingHTTPPort, r.(*jReceiver).config.AgentHTTPPort, "agent http port should be default")
}

func TestAgentRemoteSamplingEndpoint(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	rCfg := cfg.(*Config)

	endpoint := "localhost:1234"
	rCfg.Protocols.ThriftCompact = &ProtocolUDP{
		Endpoint: defaultThriftCompactBindEndpoint,
	}
	rCfg.RemoteSampling = &RemoteSamplingConfig{
		GRPCClientSettings: configgrpc.GRPCClientSettings{
			Endpoint: endpoint,
		},
	}
	set := componenttest.NewNopReceiverCreateSettings()
	r, err := factory.CreateTracesReceiver(context.Background(), set, cfg, nil)

	assert.NoError(t, err, "create trace receiver should not error")
	assert.Equal(t, endpoint, r.(*jReceiver).config.RemoteSamplingClientSettings.Endpoint)
	assert.Equal(t, defaultAgentRemoteSamplingHTTPPort, r.(*jReceiver).config.AgentHTTPPort, "agent http port should be default")
}

func TestCreateNoPort(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	cfg.(*Config).Protocols.ThriftHTTP = &confighttp.HTTPServerSettings{
		Endpoint: "localhost:",
	}
	set := componenttest.NewNopReceiverCreateSettings()
	_, err := factory.CreateTracesReceiver(context.Background(), set, cfg, nil)
	assert.Error(t, err, "receiver creation with no port number must fail")
}

func TestCreateLargePort(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	cfg.(*Config).Protocols.ThriftHTTP = &confighttp.HTTPServerSettings{
		Endpoint: "localhost:65536",
	}
	set := componenttest.NewNopReceiverCreateSettings()
	_, err := factory.CreateTracesReceiver(context.Background(), set, cfg, nil)
	assert.Error(t, err, "receiver creation with too large port number must fail")
}

func TestCreateInvalidHost(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	cfg.(*Config).Protocols.GRPC = &configgrpc.GRPCServerSettings{
		NetAddr: confignet.NetAddr{
			Endpoint:  "1234",
			Transport: "tcp",
		},
	}

	set := componenttest.NewNopReceiverCreateSettings()
	_, err := factory.CreateTracesReceiver(context.Background(), set, cfg, nil)
	assert.Error(t, err, "receiver creation with bad hostname must fail")
}

func TestCreateNoProtocols(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	cfg.(*Config).Protocols = Protocols{}
	set := componenttest.NewNopReceiverCreateSettings()
	_, err := factory.CreateTracesReceiver(context.Background(), set, cfg, nil)
	assert.Error(t, err, "receiver creation with no protocols must fail")
}

func TestThriftBinaryBadPort(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	cfg.(*Config).Protocols.ThriftBinary = &ProtocolUDP{
		Endpoint: "localhost:65536",
	}
	set := componenttest.NewNopReceiverCreateSettings()
	_, err := factory.CreateTracesReceiver(context.Background(), set, cfg, nil)
	assert.Error(t, err, "receiver creation with a bad thrift binary port must fail")
}

func TestThriftCompactBadPort(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	cfg.(*Config).Protocols.ThriftCompact = &ProtocolUDP{
		Endpoint: "localhost:65536",
	}

	set := componenttest.NewNopReceiverCreateSettings()
	_, err := factory.CreateTracesReceiver(context.Background(), set, cfg, nil)
	assert.Error(t, err, "receiver creation with a bad thrift compact port must fail")
}

func TestRemoteSamplingConfigPropagation(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	rCfg := cfg.(*Config)

	hostPort := 5778
	endpoint := "localhost:1234"
	strategyFile := "strategies.json"
	rCfg.Protocols.GRPC = &configgrpc.GRPCServerSettings{
		NetAddr: confignet.NetAddr{
			Endpoint:  defaultGRPCBindEndpoint,
			Transport: "tcp",
		},
	}
	rCfg.RemoteSampling = &RemoteSamplingConfig{
		GRPCClientSettings: configgrpc.GRPCClientSettings{
			Endpoint: endpoint,
		},
		HostEndpoint: fmt.Sprintf("localhost:%d", hostPort),
		StrategyFile: strategyFile,
	}
	set := componenttest.NewNopReceiverCreateSettings()
	r, err := factory.CreateTracesReceiver(context.Background(), set, cfg, nil)

	assert.NoError(t, err, "create trace receiver should not error")
	assert.Equal(t, endpoint, r.(*jReceiver).config.RemoteSamplingClientSettings.Endpoint)
	assert.Equal(t, hostPort, r.(*jReceiver).config.AgentHTTPPort, "agent http port should be configured value")
	assert.Equal(t, strategyFile, r.(*jReceiver).config.RemoteSamplingStrategyFile)
}

func TestRemoteSamplingFileRequiresGRPC(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	rCfg := cfg.(*Config)

	// Remove all default protocols
	rCfg.Protocols = Protocols{}
	rCfg.Protocols.ThriftCompact = &ProtocolUDP{
		Endpoint: defaultThriftCompactBindEndpoint,
	}
	rCfg.RemoteSampling = &RemoteSamplingConfig{
		StrategyFile: "strategies.json",
	}
	set := componenttest.NewNopReceiverCreateSettings()
	_, err := factory.CreateTracesReceiver(context.Background(), set, cfg, nil)

	assert.Error(t, err, "create trace receiver should error")
}
