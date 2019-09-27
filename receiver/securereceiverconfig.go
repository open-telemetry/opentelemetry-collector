// Copyright 2019 OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package receiver

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
)

// SecureReceiverSettings defines common settings for receivers that use Transport Layer Security (TLS)
type SecureReceiverSettings struct {
	configmodels.ReceiverSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
	// Configures the receiver to use TLS.
	// The default value is nil, which will cause the receiver to not use TLS.
	TLSCredentials *TLSCredentials `mapstructure:"tls_credentials, omitempty"`
}

// TLSCredentials contains path information for a certificate and key to be used for TLS
type TLSCredentials struct {
	// CertFile is the file path containing the TLS certificate.
	CertFile string `mapstructure:"cert_file"`

	// KeyFile is the file path containing the TLS key.
	KeyFile string `mapstructure:"key_file"`
}

// ToGrpcServerOption creates a gRPC ServerOption from TLSCredentials. If TLSCredentials is nil, returns empty option.
func (tlsCreds *TLSCredentials) ToGrpcServerOption() (opt grpc.ServerOption, err error) {
	if tlsCreds == nil {
		return grpc.EmptyServerOption{}, nil
	}

	transportCreds, err := credentials.NewServerTLSFromFile(tlsCreds.CertFile, tlsCreds.KeyFile)
	if err != nil {
		return nil, err
	}
	gRPCCredsOpt := grpc.Creds(transportCreds)
	return gRPCCredsOpt, nil
}
