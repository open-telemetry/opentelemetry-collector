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

	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/keepalive"

	"go.opentelemetry.io/collector/config/configtls"
)

// Compression gRPC keys for supported compression types within collector
const (
	CompressionUnsupported = ""
	CompressionGzip        = "gzip"
)

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
	KeepaliveParameters *KeepaliveConfig `mapstructure:"keepalive"`

	// WaitForReady parameter configures client to wait for ready state before sending data.
	// (https://github.com/grpc/grpc/blob/master/doc/wait-for-ready.md)
	WaitForReady bool `mapstructure:"wait_for_ready"`
}

// KeepaliveConfig exposes the keepalive.ClientParameters to be used by the exporter.
// Refer to the original data-structure for the meaning of each parameter.
type KeepaliveConfig struct {
	Time                time.Duration `mapstructure:"time,omitempty"`
	Timeout             time.Duration `mapstructure:"timeout,omitempty"`
	PermitWithoutStream bool          `mapstructure:"permit_without_stream,omitempty"`
}

// GrpcSettingsToDialOptions maps configgrpc.GRPCClientSettings to a slice of dial options for gRPC
func GrpcSettingsToDialOptions(settings GRPCClientSettings) ([]grpc.DialOption, error) {
	opts := []grpc.DialOption{}

	if settings.Compression != "" {
		if compressionKey := GetGRPCCompressionKey(settings.Compression); compressionKey != CompressionUnsupported {
			opts = append(opts, grpc.WithDefaultCallOptions(grpc.UseCompressor(compressionKey)))
		} else {
			return nil, fmt.Errorf("unsupported compression type %q", settings.Compression)
		}
	}

	tlsDialOption, err := settings.TLSSetting.LoadGRPCTLSCredentials()
	if err != nil {
		return nil, err
	}
	opts = append(opts, tlsDialOption)

	if settings.KeepaliveParameters != nil {
		keepAliveOption := grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                settings.KeepaliveParameters.Time,
			Timeout:             settings.KeepaliveParameters.Timeout,
			PermitWithoutStream: settings.KeepaliveParameters.PermitWithoutStream,
		})
		opts = append(opts, keepAliveOption)
	}

	return opts, nil
}

var (
	// Map of opentelemetry compression types to grpc registered compression types
	grpcCompressionKeyMap = map[string]string{
		CompressionGzip: gzip.Name,
	}
)

// GetGRPCCompressionKey returns the grpc registered compression key if the
// passed in compression key is supported, and CompressionUnsupported otherwise
func GetGRPCCompressionKey(compressionType string) string {
	compressionKey := strings.ToLower(compressionType)
	if encodingKey, ok := grpcCompressionKeyMap[compressionKey]; ok {
		return encodingKey
	}
	return CompressionUnsupported
}
