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
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configmodels"
)

const (
	// The config field name to load the protocol map from
	protocolsFieldName = "protocols"

	// Default UDP server options
	defaultQueueSize        = 1_000
	defaultMaxPacketSize    = 65_000
	defaultServerWorkers    = 10
	defaultSocketBufferSize = 0
)

// RemoteSamplingConfig defines config key for remote sampling fetch endpoint
type RemoteSamplingConfig struct {
	HostEndpoint                  string `mapstructure:"host_endpoint"`
	StrategyFile                  string `mapstructure:"strategy_file"`
	configgrpc.GRPCClientSettings `mapstructure:",squash"`
}

type Protocols struct {
	GRPC          *configgrpc.GRPCServerSettings `mapstructure:"grpc"`
	ThriftHTTP    *confighttp.HTTPServerSettings `mapstructure:"thrift_http"`
	ThriftBinary  *ProtocolUDP                   `mapstructure:"thrift_binary"`
	ThriftCompact *ProtocolUDP                   `mapstructure:"thrift_compact"`
}

type ProtocolUDP struct {
	Endpoint        string `mapstructure:"endpoint"`
	ServerConfigUDP `mapstructure:",squash"`
}

type ServerConfigUDP struct {
	QueueSize        int `mapstructure:"queue_size"`
	MaxPacketSize    int `mapstructure:"max_packet_size"`
	Workers          int `mapstructure:"workers"`
	SocketBufferSize int `mapstructure:"socket_buffer_size"`
}

func DefaultServerConfigUDP() ServerConfigUDP {
	return ServerConfigUDP{
		QueueSize:        defaultQueueSize,
		MaxPacketSize:    defaultMaxPacketSize,
		Workers:          defaultServerWorkers,
		SocketBufferSize: defaultSocketBufferSize,
	}
}

// Config defines configuration for Jaeger receiver.
type Config struct {
	configmodels.ReceiverSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
	Protocols                     `mapstructure:"protocols"`
	RemoteSampling                *RemoteSamplingConfig `mapstructure:"remote_sampling"`
}
