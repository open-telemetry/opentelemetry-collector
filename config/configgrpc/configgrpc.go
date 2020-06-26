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

// Package configgrpc defines the gRPC configuration settings.
package configgrpc

import (
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/keepalive"

	"go.opentelemetry.io/collector/config/configprotocol"
	"go.opentelemetry.io/collector/config/configtls"
)

// Compression gRPC keys for supported compression types within collector
const (
	CompressionUnsupported = ""
	CompressionGzip        = "gzip"
)

var (
	// Map of opentelemetry compression types to grpc registered compression types
	grpcCompressionKeyMap = map[string]string{
		CompressionGzip: gzip.Name,
	}
)

// KeepaliveClientConfig exposes the keepalive.ClientParameters to be used by the exporter.
// Refer to the original data-structure for the meaning of each parameter:
// https://godoc.org/google.golang.org/grpc/keepalive#ClientParameters
type KeepaliveClientConfig struct {
	Time                time.Duration `mapstructure:"time,omitempty"`
	Timeout             time.Duration `mapstructure:"timeout,omitempty"`
	PermitWithoutStream bool          `mapstructure:"permit_without_stream,omitempty"`
}

// GRPCClientSettings defines common settings for a gRPC client configuration.
type GRPCClientSettings struct {
	// The headers associated with gRPC requests.
	Headers map[string]string `mapstructure:"headers"`

	// The target to which the exporter is going to send traces or metrics,
	// using the gRPC protocol. The valid syntax is described at
	// https://github.com/grpc/grpc/blob/master/doc/naming.md.
	Endpoint string `mapstructure:"endpoint"`

	// The compression key for supported compression types within
	// collector. Currently the only supported mode is `gzip`.
	Compression string `mapstructure:"compression"`

	// TLSSetting struct exposes TLS client configuration.
	TLSSetting configtls.TLSClientSetting `mapstructure:",squash"`

	// The keepalive parameters for client gRPC. See grpc.WithKeepaliveParams
	// (https://godoc.org/google.golang.org/grpc#WithKeepaliveParams).
	Keepalive *KeepaliveClientConfig `mapstructure:"keepalive"`

	// WaitForReady parameter configures client to wait for ready state before sending data.
	// (https://github.com/grpc/grpc/blob/master/doc/wait-for-ready.md)
	WaitForReady bool `mapstructure:"wait_for_ready"`
}

type KeepaliveServerConfig struct {
	ServerParameters  *KeepaliveServerParameters  `mapstructure:"server_parameters,omitempty"`
	EnforcementPolicy *KeepaliveEnforcementPolicy `mapstructure:"enforcement_policy,omitempty"`
}

// KeepaliveServerParameters allow configuration of the keepalive.ServerParameters.
// The same default values as keepalive.ServerParameters are applicable and get applied by the server.
// See https://godoc.org/google.golang.org/grpc/keepalive#ServerParameters for details.
type KeepaliveServerParameters struct {
	MaxConnectionIdle     time.Duration `mapstructure:"max_connection_idle,omitempty"`
	MaxConnectionAge      time.Duration `mapstructure:"max_connection_age,omitempty"`
	MaxConnectionAgeGrace time.Duration `mapstructure:"max_connection_age_grace,omitempty"`
	Time                  time.Duration `mapstructure:"time,omitempty"`
	Timeout               time.Duration `mapstructure:"timeout,omitempty"`
}

// KeepaliveEnforcementPolicy allow configuration of the keepalive.EnforcementPolicy.
// The same default values as keepalive.EnforcementPolicy are applicable and get applied by the server.
// See https://godoc.org/google.golang.org/grpc/keepalive#EnforcementPolicy for details.
type KeepaliveEnforcementPolicy struct {
	MinTime             time.Duration `mapstructure:"min_time,omitempty"`
	PermitWithoutStream bool          `mapstructure:"permit_without_stream,omitempty"`
}

type GRPCServerSettings struct {
	// Configures the generic server protocol.
	configprotocol.ProtocolServerSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct

	// MaxRecvMsgSizeMiB sets the maximum size (in MiB) of messages accepted by the server.
	MaxRecvMsgSizeMiB uint64 `mapstructure:"max_recv_msg_size_mib,omitempty"`

	// MaxConcurrentStreams sets the limit on the number of concurrent streams to each ServerTransport.
	// It has effect only for streaming RPCs.
	MaxConcurrentStreams uint32 `mapstructure:"max_concurrent_streams,omitempty"`

	// Keepalive anchor for all the settings related to keepalive.
	Keepalive *KeepaliveServerConfig `mapstructure:"keepalive,omitempty"`
}

// ToServerOption maps configgrpc.GRPCClientSettings to a slice of dial options for gRPC
func (settings *GRPCClientSettings) ToDialOptions() ([]grpc.DialOption, error) {
	opts := []grpc.DialOption{}

	if settings.Compression != "" {
		if compressionKey := GetGRPCCompressionKey(settings.Compression); compressionKey != CompressionUnsupported {
			opts = append(opts, grpc.WithDefaultCallOptions(grpc.UseCompressor(compressionKey)))
		} else {
			return nil, fmt.Errorf("unsupported compression type %q", settings.Compression)
		}
	}

	tlsCfg, err := settings.TLSSetting.LoadTLSConfig()
	if err != nil {
		return nil, err
	}
	tlsDialOption := grpc.WithInsecure()
	if tlsCfg != nil {
		tlsDialOption = grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg))
	}
	opts = append(opts, tlsDialOption)

	if settings.Keepalive != nil {
		keepAliveOption := grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                settings.Keepalive.Time,
			Timeout:             settings.Keepalive.Timeout,
			PermitWithoutStream: settings.Keepalive.PermitWithoutStream,
		})
		opts = append(opts, keepAliveOption)
	}

	return opts, nil
}

// ToServerOption maps configgrpc.GRPCServerSettings to a slice of server options for gRPC
func (gss *GRPCServerSettings) ToServerOption() ([]grpc.ServerOption, error) {
	var opts []grpc.ServerOption

	if gss.TLSCredentials != nil {
		tlsCfg, err := gss.TLSCredentials.LoadTLSConfig()
		if err != nil {
			return nil, errors.Wrap(err, "error initializing TLS Credentials")
		}
		opts = append(opts, grpc.Creds(credentials.NewTLS(tlsCfg)))
	}

	if gss.MaxRecvMsgSizeMiB > 0 {
		opts = append(opts, grpc.MaxRecvMsgSize(int(gss.MaxRecvMsgSizeMiB*1024*1024)))
	}

	if gss.MaxConcurrentStreams > 0 {
		opts = append(opts, grpc.MaxConcurrentStreams(gss.MaxConcurrentStreams))
	}

	// The default values referenced in the GRPC docs are set within the server, so this code doesn't need
	// to apply them over zero/nil values before passing these as grpc.ServerOptions.
	// The following shows the server code for applying default grpc.ServerOptions.
	// https://github.com/grpc/grpc-go/blob/120728e1f775e40a2a764341939b78d666b08260/internal/transport/http2_server.go#L184-L200
	if gss.Keepalive != nil {
		if gss.Keepalive.ServerParameters != nil {
			svrParams := gss.Keepalive.ServerParameters
			opts = append(opts, grpc.KeepaliveParams(keepalive.ServerParameters{
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
		// https://github.com/grpc/grpc-go/blob/120728e1f775e40a2a764341939b78d666b08260/internal/transport/http2_server.go#L202-L205
		if gss.Keepalive.EnforcementPolicy != nil {
			enfPol := gss.Keepalive.EnforcementPolicy
			opts = append(opts, grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
				MinTime:             enfPol.MinTime,
				PermitWithoutStream: enfPol.PermitWithoutStream,
			}))
		}
	}

	return opts, nil
}

// GetGRPCCompressionKey returns the grpc registered compression key if the
// passed in compression key is supported, and CompressionUnsupported otherwise
func GetGRPCCompressionKey(compressionType string) string {
	compressionKey := strings.ToLower(compressionType)
	if encodingKey, ok := grpcCompressionKeyMap[compressionKey]; ok {
		return encodingKey
	}
	return CompressionUnsupported
}
