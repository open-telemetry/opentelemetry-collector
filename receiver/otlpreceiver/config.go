// Copyright The OpenTelemetry Authors
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

package otlpreceiver

import (
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/configmodels"
)

// Config defines configuration for OTLP receiver.
type Config struct {
	configmodels.ReceiverSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct

	// Configures the receiver server protocol.
	configgrpc.GRPCServerSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct

	// Transport to use: one of tcp or unix, defaults to tcp
	Transport string `mapstructure:"transport"`

	// CorsOrigins are the allowed CORS origins for HTTP/JSON requests to grpc-gateway adapter
	// for the OTLP receiver. See github.com/rs/cors
	// An empty list means that CORS is not enabled at all. A wildcard (*) can be
	// used to match any origin or one or more characters of an origin.
	CorsOrigins []string `mapstructure:"cors_allowed_origins"`
}

func (rOpts *Config) buildOptions() ([]Option, error) {
	var opts []Option
	if len(rOpts.CorsOrigins) > 0 {
		opts = append(opts, WithCorsOrigins(rOpts.CorsOrigins))
	}

	grpcServerOptions, err := rOpts.GRPCServerSettings.ToServerOption()
	if err != nil {
		return nil, err
	}
	if len(grpcServerOptions) > 0 {
		opts = append(opts, WithGRPCServerOptions(grpcServerOptions...))
	}

	return opts, nil
}
