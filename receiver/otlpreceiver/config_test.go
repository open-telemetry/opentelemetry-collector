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

package otlpreceiver

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/confmaptest"
)

func TestUnmarshalDefaultConfig(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "default.yaml"))
	require.NoError(t, err)
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.NoError(t, component.UnmarshalConfig(cm, cfg))
	assert.Equal(t, factory.CreateDefaultConfig(), cfg)
}

func TestUnmarshalConfigOnlyGRPC(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "only_grpc.yaml"))
	require.NoError(t, err)
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.NoError(t, component.UnmarshalConfig(cm, cfg))

	defaultOnlyGRPC := factory.CreateDefaultConfig().(*Config)
	defaultOnlyGRPC.HTTP = nil
	assert.Equal(t, defaultOnlyGRPC, cfg)
}

func TestUnmarshalConfigOnlyHTTP(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "only_http.yaml"))
	require.NoError(t, err)
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.NoError(t, component.UnmarshalConfig(cm, cfg))

	defaultOnlyHTTP := factory.CreateDefaultConfig().(*Config)
	defaultOnlyHTTP.GRPC = nil
	assert.Equal(t, defaultOnlyHTTP, cfg)
}

func TestUnmarshalConfigOnlyHTTPNull(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "only_http_null.yaml"))
	require.NoError(t, err)
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.NoError(t, component.UnmarshalConfig(cm, cfg))

	defaultOnlyHTTP := factory.CreateDefaultConfig().(*Config)
	defaultOnlyHTTP.GRPC = nil
	assert.Equal(t, defaultOnlyHTTP, cfg)
}

func TestUnmarshalConfigOnlyHTTPEmptyMap(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "only_http_empty_map.yaml"))
	require.NoError(t, err)
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.NoError(t, component.UnmarshalConfig(cm, cfg))

	defaultOnlyHTTP := factory.CreateDefaultConfig().(*Config)
	defaultOnlyHTTP.GRPC = nil
	assert.Equal(t, defaultOnlyHTTP, cfg)
}

func TestUnmarshalConfig(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config.yaml"))
	require.NoError(t, err)
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.NoError(t, component.UnmarshalConfig(cm, cfg))
	assert.Equal(t,
		&Config{
			Protocols: Protocols{
				GRPC: &configgrpc.GRPCServerSettings{
					NetAddr: confignet.NetAddr{
						Endpoint:  "0.0.0.0:4317",
						Transport: "tcp",
					},
					TLSSetting: &configtls.TLSServerSetting{
						TLSSetting: configtls.TLSSetting{
							CertFile: "test.crt",
							KeyFile:  "test.key",
						},
					},
					MaxRecvMsgSizeMiB:    32,
					MaxConcurrentStreams: 16,
					ReadBufferSize:       1024,
					WriteBufferSize:      1024,
					Keepalive: &configgrpc.KeepaliveServerConfig{
						ServerParameters: &configgrpc.KeepaliveServerParameters{
							MaxConnectionIdle:     11 * time.Second,
							MaxConnectionAge:      12 * time.Second,
							MaxConnectionAgeGrace: 13 * time.Second,
							Time:                  30 * time.Second,
							Timeout:               5 * time.Second,
						},
						EnforcementPolicy: &configgrpc.KeepaliveEnforcementPolicy{
							MinTime:             10 * time.Second,
							PermitWithoutStream: true,
						},
					},
				},
				HTTP: &confighttp.HTTPServerSettings{
					Endpoint: "0.0.0.0:4318",
					TLSSetting: &configtls.TLSServerSetting{
						TLSSetting: configtls.TLSSetting{
							CertFile: "test.crt",
							KeyFile:  "test.key",
						},
					},
					CORS: &confighttp.CORSSettings{
						AllowedOrigins: []string{"https://*.test.com", "https://test.com"},
						MaxAge:         7200,
					},
				},
			},
		}, cfg)

}

func TestUnmarshalConfigUnix(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "uds.yaml"))
	require.NoError(t, err)
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.NoError(t, component.UnmarshalConfig(cm, cfg))
	assert.Equal(t,
		&Config{
			Protocols: Protocols{
				GRPC: &configgrpc.GRPCServerSettings{
					NetAddr: confignet.NetAddr{
						Endpoint:  "/tmp/grpc_otlp.sock",
						Transport: "unix",
					},
					ReadBufferSize: 512 * 1024,
				},
				HTTP: &confighttp.HTTPServerSettings{
					Endpoint: "/tmp/http_otlp.sock",
					// Transport: "unix",
				},
			},
		}, cfg)
}

func TestUnmarshalConfigTypoDefaultProtocol(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "typo_default_proto_config.yaml"))
	require.NoError(t, err)
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.EqualError(t, component.UnmarshalConfig(cm, cfg), "1 error(s) decoding:\n\n* 'protocols' has invalid keys: htttp")
}

func TestUnmarshalConfigInvalidProtocol(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "bad_proto_config.yaml"))
	require.NoError(t, err)
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.EqualError(t, component.UnmarshalConfig(cm, cfg), "1 error(s) decoding:\n\n* 'protocols' has invalid keys: thrift")
}

func TestUnmarshalConfigEmptyProtocols(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "bad_no_proto_config.yaml"))
	require.NoError(t, err)
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.NoError(t, component.UnmarshalConfig(cm, cfg))
	assert.EqualError(t, component.ValidateConfig(cfg), "must specify at least one protocol when using the OTLP receiver")
}

func TestUnmarshalConfigEmpty(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.NoError(t, component.UnmarshalConfig(confmap.New(), cfg))
	assert.EqualError(t, component.ValidateConfig(cfg), "must specify at least one protocol when using the OTLP receiver")
}
