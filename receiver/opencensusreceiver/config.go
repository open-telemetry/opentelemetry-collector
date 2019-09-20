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

package opencensusreceiver

import (
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	"github.com/open-telemetry/opentelemetry-service/receiver"
)

// Config defines configuration for OpenCensus receiver.
type Config struct {
	receiver.SecureReceiverSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
	// CorsOrigins are the allowed CORS origins for HTTP/JSON requests to grpc-gateway adapter
	// for the OpenCensus receiver. See github.com/rs/cors
	// An empty list means that CORS is not enabled at all. A wildcard (*) can be
	// used to match any origin or one or more characters of an origin.
	CorsOrigins []string `mapstructure:"cors_allowed_origins"`

	// Keepalive anchor for all the settings related to keepalive.
	Keepalive *serverParametersAndEnforcementPolicy `mapstructure:"keepalive,omitempty"`

	// MaxRecvMsgSizeMiB sets the maximum size (in MiB) of messages accepted by the server.
	MaxRecvMsgSizeMiB uint64 `mapstructure:"max_recv_msg_size_mib,omitempty"`

	// MaxConcurrentStreams sets the limit on the number of concurrent streams to each ServerTransport.
	MaxConcurrentStreams uint32 `mapstructure:"max_concurrent_streams,omitempty"`
}

type serverParametersAndEnforcementPolicy struct {
	ServerParameters  *keepaliveServerParameters  `mapstructure:"server_parameters,omitempty"`
	EnforcementPolicy *keepaliveEnforcementPolicy `mapstructure:"enforcement_policy,omitempty"`
}

// keepaliveServerParameters allow configuration of the keepalive.ServerParameters.
// The same default values as keepalive.ServerParameters are applicable and get applied by the server.
// See https://godoc.org/google.golang.org/grpc/keepalive#ServerParameters for details.
type keepaliveServerParameters struct {
	MaxConnectionIdle     time.Duration `mapstructure:"max_connection_idle,omitempty"`
	MaxConnectionAge      time.Duration `mapstructure:"max_connection_age,omitempty"`
	MaxConnectionAgeGrace time.Duration `mapstructure:"max_connection_age_grace,omitempty"`
	Time                  time.Duration `mapstructure:"time,omitempty"`
	Timeout               time.Duration `mapstructure:"timeout,omitempty"`
}

// keepaliveEnforcementPolicy allow configuration of the keepalive.EnforcementPolicy.
// The same default values as keepalive.EnforcementPolicy are applicable and get applied by the server.
// See https://godoc.org/google.golang.org/grpc/keepalive#EnforcementPolicy for details.
type keepaliveEnforcementPolicy struct {
	MinTime             time.Duration `mapstructure:"min_time,omitempty"`
	PermitWithoutStream bool          `mapstructure:"permit_without_stream,omitempty"`
}

func (rOpts *Config) buildOptions() (opts []Option, err error) {
	tlsCredsOption, hasTLSCreds, err := ToOpenCensusReceiverServerOption(rOpts.TLSCredentials)
	if err != nil {
		return opts, fmt.Errorf("error initializing OpenCensus receiver %q TLS Credentials: %v", rOpts.NameVal, err)
	}
	if hasTLSCreds {
		opts = append(opts, tlsCredsOption)
	}
	if len(rOpts.CorsOrigins) > 0 {
		opts = append(opts, WithCorsOrigins(rOpts.CorsOrigins))
	}

	grpcServerOptions := rOpts.grpcServerOptions()
	if len(grpcServerOptions) > 0 {
		opts = append(opts, WithGRPCServerOptions(grpcServerOptions...))
	}

	return opts, err
}

func (rOpts *Config) grpcServerOptions() []grpc.ServerOption {
	var grpcServerOptions []grpc.ServerOption
	if rOpts.MaxRecvMsgSizeMiB > 0 {
		grpcServerOptions = append(grpcServerOptions, grpc.MaxRecvMsgSize(int(rOpts.MaxRecvMsgSizeMiB*1024*1024)))
	}
	if rOpts.MaxConcurrentStreams > 0 {
		grpcServerOptions = append(grpcServerOptions, grpc.MaxConcurrentStreams(rOpts.MaxConcurrentStreams))
	}
	// The default values referenced in the GRPC docs are set within the server, so this code doesn't need
	// to apply them over zero/nil values before passing these as grpc.ServerOptions.
	// The following shows the server code for applying default grpc.ServerOptions.
	// https://sourcegraph.com/github.com/grpc/grpc-go@120728e1f775e40a2a764341939b78d666b08260/-/blob/internal/transport/http2_server.go#L184-200
	if rOpts.Keepalive != nil {
		if rOpts.Keepalive.ServerParameters != nil {
			svrParams := rOpts.Keepalive.ServerParameters
			grpcServerOptions = append(grpcServerOptions, grpc.KeepaliveParams(keepalive.ServerParameters{
				MaxConnectionIdle:     svrParams.MaxConnectionIdle,
				MaxConnectionAge:      svrParams.MaxConnectionAge,
				MaxConnectionAgeGrace: svrParams.MaxConnectionAgeGrace,
				Time:                  svrParams.Time,
				Timeout:               svrParams.Timeout,
			}))
		}
		// The default values referenced in the GRPC are set within the server, so this code doesn't need
		// to apply them over zero/nil values before passing these as grpc.ServerOptions.
		// The following shows the server code for applying default grpc.ServerOptions.
		// https://sourcegraph.com/github.com/grpc/grpc-go@120728e1f775e40a2a764341939b78d666b08260/-/blob/internal/transport/http2_server.go#L202-205
		if rOpts.Keepalive.EnforcementPolicy != nil {
			enfPol := rOpts.Keepalive.EnforcementPolicy
			grpcServerOptions = append(grpcServerOptions, grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
				MinTime:             enfPol.MinTime,
				PermitWithoutStream: enfPol.PermitWithoutStream,
			}))
		}
	}

	return grpcServerOptions
}

// ToOpenCensusReceiverServerOption checks if the TLS credentials
// in the form of a certificate file and a key file. If they aren't,
// it will return opencensusreceiver.WithNoopOption() and a nil error.
// Otherwise, it will try to retrieve gRPC transport credentials from the file combinations,
// and create a option, along with any errors encountered while retrieving the credentials.
func ToOpenCensusReceiverServerOption(tlsCreds *receiver.TLSCredentials) (opt Option, ok bool, err error) {
	gRPCCredsOpt, err := tlsCreds.ToGrpcServerOption()
	if err != nil {
		return nil, false, err
	}
	return WithGRPCServerOptions(gRPCCredsOpt), true, nil
}
