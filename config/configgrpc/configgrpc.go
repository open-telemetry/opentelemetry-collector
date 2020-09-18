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

// Package configgrpc defines the gRPC configuration settings.
package configgrpc

import (
	"fmt"
	"net"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer/roundrobin"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/keepalive"

	"go.opentelemetry.io/collector/config/configauth"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/config/configtls"
)

// Compression gRPC keys for supported compression types within collector
const (
	CompressionUnsupported = ""
	CompressionGzip        = "gzip"

	PerRPCAuthTypeBearer = "bearer"
)

var (
	// Map of opentelemetry compression types to grpc registered compression types
	grpcCompressionKeyMap = map[string]string{
		CompressionGzip: gzip.Name,
	}
)

// Allowed balancer names to be set in grpclb_policy to discover the servers
var allowedBalancerNames = []string{roundrobin.Name, grpc.PickFirstBalancerName}

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
	// The target to which the exporter is going to send traces or metrics,
	// using the gRPC protocol. The valid syntax is described at
	// https://github.com/grpc/grpc/blob/master/doc/naming.md.
	Endpoint string `mapstructure:"endpoint"`

	// The compression key for supported compression types within
	// collector. Currently the only supported mode is `gzip`.
	Compression string `mapstructure:"compression"`

	// TLSSetting struct exposes TLS client configuration.
	TLSSetting configtls.TLSClientSetting `mapstructure:",squash"`

	// The keepalive parameters for gRPC client. See grpc.WithKeepaliveParams
	// (https://godoc.org/google.golang.org/grpc#WithKeepaliveParams).
	Keepalive *KeepaliveClientConfig `mapstructure:"keepalive"`

	// ReadBufferSize for gRPC client. See grpc.WithReadBufferSize
	// (https://godoc.org/google.golang.org/grpc#WithReadBufferSize).
	ReadBufferSize int `mapstructure:"read_buffer_size"`

	// WriteBufferSize for gRPC gRPC. See grpc.WithWriteBufferSize
	// (https://godoc.org/google.golang.org/grpc#WithWriteBufferSize).
	WriteBufferSize int `mapstructure:"write_buffer_size"`

	// WaitForReady parameter configures client to wait for ready state before sending data.
	// (https://github.com/grpc/grpc/blob/master/doc/wait-for-ready.md)
	WaitForReady bool `mapstructure:"wait_for_ready"`

	// The headers associated with gRPC requests.
	Headers map[string]string `mapstructure:"headers"`

	// PerRPCAuth parameter configures the client to send authentication data on a per-RPC basis.
	PerRPCAuth *PerRPCAuthConfig `mapstructure:"per_rpc_auth"`

	// Sets the balancer in grpclb_policy to discover the servers. Default is pick_first
	// https://github.com/grpc/grpc-go/blob/master/examples/features/load_balancing/README.md
	BalancerName string `mapstructure:"balancer_name"`
}

type KeepaliveServerConfig struct {
	ServerParameters  *KeepaliveServerParameters  `mapstructure:"server_parameters,omitempty"`
	EnforcementPolicy *KeepaliveEnforcementPolicy `mapstructure:"enforcement_policy,omitempty"`
}

// PerRPCAuthConfig specifies how the Per-RPC authentication data should be obtained.
type PerRPCAuthConfig struct {
	// AuthType represents the authentication type to use. Currently, only 'bearer' is supported.
	AuthType string `mapstructure:"type,omitempty"`

	// BearerToken specifies the bearer token to use for every RPC.
	BearerToken string `mapstructure:"bearer_token,omitempty"`
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
	// Server net.Addr config. For transport only "tcp" and "unix" are valid options.
	NetAddr confignet.NetAddr `mapstructure:",squash"`

	// Configures the protocol to use TLS.
	// The default value is nil, which will cause the protocol to not use TLS.
	TLSSetting *configtls.TLSServerSetting `mapstructure:"tls_settings,omitempty"`

	// MaxRecvMsgSizeMiB sets the maximum size (in MiB) of messages accepted by the server.
	MaxRecvMsgSizeMiB uint64 `mapstructure:"max_recv_msg_size_mib"`

	// MaxConcurrentStreams sets the limit on the number of concurrent streams to each ServerTransport.
	// It has effect only for streaming RPCs.
	MaxConcurrentStreams uint32 `mapstructure:"max_concurrent_streams"`

	// ReadBufferSize for gRPC server. See grpc.ReadBufferSize
	// (https://godoc.org/google.golang.org/grpc#ReadBufferSize).
	ReadBufferSize int `mapstructure:"read_buffer_size"`

	// WriteBufferSize for gRPC server. See grpc.WriteBufferSize
	// (https://godoc.org/google.golang.org/grpc#WriteBufferSize).
	WriteBufferSize int `mapstructure:"write_buffer_size"`

	// Keepalive anchor for all the settings related to keepalive.
	Keepalive *KeepaliveServerConfig `mapstructure:"keepalive,omitempty"`

	// Auth for this receiver
	Auth *configauth.Authentication `mapstructure:"auth,omitempty"`
}

// ToDialOptions maps configgrpc.GRPCClientSettings to a slice of dial options for gRPC
func (gcs *GRPCClientSettings) ToDialOptions() ([]grpc.DialOption, error) {
	var opts []grpc.DialOption
	if gcs.Compression != "" {
		if compressionKey := GetGRPCCompressionKey(gcs.Compression); compressionKey != CompressionUnsupported {
			opts = append(opts, grpc.WithDefaultCallOptions(grpc.UseCompressor(compressionKey)))
		} else {
			return nil, fmt.Errorf("unsupported compression type %q", gcs.Compression)
		}
	}

	tlsCfg, err := gcs.TLSSetting.LoadTLSConfig()
	if err != nil {
		return nil, err
	}
	tlsDialOption := grpc.WithInsecure()
	if tlsCfg != nil {
		tlsDialOption = grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg))
	}
	opts = append(opts, tlsDialOption)

	if gcs.ReadBufferSize > 0 {
		opts = append(opts, grpc.WithReadBufferSize(gcs.ReadBufferSize))
	}

	if gcs.WriteBufferSize > 0 {
		opts = append(opts, grpc.WithWriteBufferSize(gcs.WriteBufferSize))
	}

	if gcs.Keepalive != nil {
		keepAliveOption := grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                gcs.Keepalive.Time,
			Timeout:             gcs.Keepalive.Timeout,
			PermitWithoutStream: gcs.Keepalive.PermitWithoutStream,
		})
		opts = append(opts, keepAliveOption)
	}

	if gcs.PerRPCAuth != nil {
		if strings.EqualFold(gcs.PerRPCAuth.AuthType, PerRPCAuthTypeBearer) {
			sToken := gcs.PerRPCAuth.BearerToken
			token := BearerToken(sToken)
			opts = append(opts, grpc.WithPerRPCCredentials(token))
		} else {
			return nil, fmt.Errorf("unsupported per-RPC auth type %q", gcs.PerRPCAuth.AuthType)
		}
	}

	if gcs.BalancerName != "" {
		valid := validateBalancerName(gcs.BalancerName)
		if !valid {
			return nil, fmt.Errorf("invalid balancer_name: %s", gcs.BalancerName)
		}
		opts = append(opts, grpc.WithDefaultServiceConfig(fmt.Sprintf(`{"loadBalancingPolicy":"%s"}`, gcs.BalancerName)))
	}

	return opts, nil
}

func validateBalancerName(balancerName string) bool {
	for _, item := range allowedBalancerNames {
		if item == balancerName {
			return true
		}
	}
	return false
}

func (gss *GRPCServerSettings) ToListener() (net.Listener, error) {
	return gss.NetAddr.Listen()
}

// ToServerOption maps configgrpc.GRPCServerSettings to a slice of server options for gRPC
func (gss *GRPCServerSettings) ToServerOption() ([]grpc.ServerOption, error) {
	var opts []grpc.ServerOption

	if gss.TLSSetting != nil {
		tlsCfg, err := gss.TLSSetting.LoadTLSConfig()
		if err != nil {
			return nil, err
		}
		opts = append(opts, grpc.Creds(credentials.NewTLS(tlsCfg)))
	}

	if gss.MaxRecvMsgSizeMiB > 0 {
		opts = append(opts, grpc.MaxRecvMsgSize(int(gss.MaxRecvMsgSizeMiB*1024*1024)))
	}

	if gss.MaxConcurrentStreams > 0 {
		opts = append(opts, grpc.MaxConcurrentStreams(gss.MaxConcurrentStreams))
	}

	if gss.ReadBufferSize > 0 {
		opts = append(opts, grpc.ReadBufferSize(gss.ReadBufferSize))
	}

	if gss.WriteBufferSize > 0 {
		opts = append(opts, grpc.WriteBufferSize(gss.WriteBufferSize))
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

	if gss.Auth != nil {
		authOpts, err := gss.Auth.ToServerOptions()
		if err != nil {
			return nil, err
		}
		opts = append(opts, authOpts...)
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
