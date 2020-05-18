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

// Package configgrpc defines the gRPC configuration settings.
package configgrpc

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/keepalive"
)

// Compression gRPC keys for supported compression types within collector
const (
	CompressionUnsupported = ""
	CompressionGzip        = "gzip"
)

// GRPCSettings defines common settings for a gRPC configuration.
type GRPCSettings struct {
	// The headers associated with gRPC requests.
	Headers map[string]string `mapstructure:"headers"`

	// The target to which the exporter is going to send traces or metrics,
	// using the gRPC protocol. The valid syntax is described at
	// https://github.com/grpc/grpc/blob/master/doc/naming.md.
	Endpoint string `mapstructure:"endpoint"`

	// The compression key for supported compression types within
	// collector. Currently the only supported mode is `gzip`.
	Compression string `mapstructure:"compression"`

	// TLSConfig struct exposes TLS client configuration.
	TLSConfig TLSConfig `mapstructure:",squash"`

	// The keepalive parameters for client gRPC. See grpc.WithKeepaliveParams
	// (https://godoc.org/google.golang.org/grpc#WithKeepaliveParams).
	KeepaliveParameters *KeepaliveConfig `mapstructure:"keepalive"`

	// WaitForReady parameter configures client to wait for ready state before sending data.
	// (https://github.com/grpc/grpc/blob/master/doc/wait-for-ready.md)
	WaitForReady bool `mapstructure:"wait_for_ready"`
}

// TLSConfig exposes client TLS configuration.
type TLSConfig struct {
	// Root CA certificate file for TLS credentials of gRPC client. Should
	// only be used if `secure` is set to true.
	CaCert string `mapstructure:"cert_pem_file"`

	// Client certificate file for TLS credentials of gRPC client.
	ClientCert string `mapstructure:"client_cert_pem_file"`

	// Client key file for TLS credentials of gRPC client.
	ClientKey string `mapstructure:"client_cert_key_file"`

	// Whether to enable client transport security for the exporter's gRPC
	// connection. See https://godoc.org/google.golang.org/grpc#WithInsecure.
	UseSecure bool `mapstructure:"secure"`

	// Authority to check against when doing TLS verification
	ServerNameOverride string `mapstructure:"server_name_override"`
}

// KeepaliveConfig exposes the keepalive.ClientParameters to be used by the exporter.
// Refer to the original data-structure for the meaning of each parameter.
type KeepaliveConfig struct {
	Time                time.Duration `mapstructure:"time,omitempty"`
	Timeout             time.Duration `mapstructure:"timeout,omitempty"`
	PermitWithoutStream bool          `mapstructure:"permit_without_stream,omitempty"`
}

// GrpcSettingsToDialOptions maps configgrpc.GRPCSettings to a slice of dial options for gRPC
func GrpcSettingsToDialOptions(settings GRPCSettings) ([]grpc.DialOption, error) {
	opts := []grpc.DialOption{}

	if settings.Compression != "" {
		if compressionKey := GetGRPCCompressionKey(settings.Compression); compressionKey != CompressionUnsupported {
			opts = append(opts, grpc.WithDefaultCallOptions(grpc.UseCompressor(compressionKey)))
		} else {
			return nil, fmt.Errorf("unsupported compression type %q", settings.Compression)
		}
	}

	if settings.TLSConfig.CaCert != "" && !settings.TLSConfig.UseSecure {
		creds, err := credentials.NewClientTLSFromFile(settings.TLSConfig.CaCert, settings.TLSConfig.ServerNameOverride)
		if err != nil {
			return nil, err
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else if settings.TLSConfig.UseSecure {
		tlsConf, err := settings.TLSConfig.LoadTLSConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS config: %w", err)
		}
		creds := credentials.NewTLS(tlsConf)
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}

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

// LoadTLSConfig loads TLS certificates and returns a tls.Config.
func (c TLSConfig) LoadTLSConfig() (*tls.Config, error) {
	var err error
	var certPool *x509.CertPool
	if len(c.CaCert) != 0 {
		// setup user specified truststore
		certPool, err = c.loadCert(c.CaCert)
		if err != nil {
			return nil, fmt.Errorf("failed to load CA CertPool: %w", err)
		}
	}
	// #nosec G402
	tlsCfg := &tls.Config{
		RootCAs:    certPool,
		ServerName: c.ServerNameOverride,
	}

	if (c.ClientCert == "" && c.ClientKey != "") || (c.ClientCert != "" && c.ClientKey == "") {
		return nil, fmt.Errorf("for client auth via TLS, either both client certificate and key must be supplied, or neither")
	}
	if c.ClientCert != "" && c.ClientKey != "" {
		tlsCert, err := tls.LoadX509KeyPair(filepath.Clean(c.ClientCert), filepath.Clean(c.ClientKey))
		if err != nil {
			return nil, fmt.Errorf("failed to load server TLS cert and key: %w", err)
		}
		tlsCfg.Certificates = append(tlsCfg.Certificates, tlsCert)
	}

	return tlsCfg, nil
}

func (c TLSConfig) loadCert(caPath string) (*x509.CertPool, error) {
	caPEM, err := ioutil.ReadFile(filepath.Clean(caPath))
	if err != nil {
		return nil, fmt.Errorf("failed to load CA %s: %w", caPath, err)
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("failed to parse CA %s", caPath)
	}
	return certPool, nil
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
