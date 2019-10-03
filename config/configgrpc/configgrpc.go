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
	"crypto/x509"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
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

	// Certificate file for TLS credentials of gRPC client. Should
	// only be used if `secure` is set to true.
	CertPemFile string `mapstructure:"cert_pem_file"`

	// Whether to enable client transport security for the exporter's gRPC
	// connection. See https://godoc.org/google.golang.org/grpc#WithInsecure.
	UseSecure bool `mapstructure:"secure"`

	// Authority to check against when doing TLS verification
	ServerNameOverride string `mapstructure:"server_name_override"`

	// The keepalive parameters for client gRPC. See grpc.WithKeepaliveParams
	// (https://godoc.org/google.golang.org/grpc#WithKeepaliveParams).
	KeepaliveParameters *KeepaliveConfig `mapstructure:"keepalive"`
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
	if settings.CertPemFile != "" {
		creds, err := credentials.NewClientTLSFromFile(settings.CertPemFile, settings.ServerNameOverride)
		if err != nil {
			return nil, err
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else if settings.UseSecure {
		certPool, err := x509.SystemCertPool()
		if err != nil {
			return nil, err
		}
		creds := credentials.NewClientTLSFromCert(certPool, settings.ServerNameOverride)
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
