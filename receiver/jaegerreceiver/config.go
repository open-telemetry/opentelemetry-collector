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
	"go.opentelemetry.io/collector/config/confignet"
)

// The config field name to load the protocol map from
const protocolsFieldName = "protocols"

// RemoteSamplingConfig defines config key for remote sampling fetch endpoint
type RemoteSamplingConfig struct {
	HostEndpoint                  string `mapstructure:"host_endpoint"`
	StrategyFile                  string `mapstructure:"strategy_file"`
	configgrpc.GRPCClientSettings `mapstructure:",squash"`
}

type Protocols struct {
	GRPC          *configgrpc.GRPCServerSettings `mapstructure:"grpc"`
	ThriftHTTP    *confighttp.HTTPServerSettings `mapstructure:"thrift_http"`
	ThriftBinary  *confignet.TCPAddr             `mapstructure:"thrift_binary"`
	ThriftCompact *confignet.TCPAddr             `mapstructure:"thrift_compact"`
}

// Config defines configuration for Jaeger receiver.
type Config struct {
	configmodels.ReceiverSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
	Protocols                     `mapstructure:"protocols"`
	RemoteSampling                *RemoteSamplingConfig `mapstructure:"remote_sampling"`
}
