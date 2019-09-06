package receiver

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/open-telemetry/opentelemetry-service/config/configmodels"
)

// SecureReceiverSettings defines common settings for receivers that use Transport Layer Security (TLS)
type SecureReceiverSettings struct {
	configmodels.ReceiverSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
	// Configures the receiver to use TLS.
	// The default value is nil, which will cause the receiver to not use TLS.
	TLSCredentials *TLSCredentials `mapstructure:"tls-credentials, omitempty"`
}

// TLSCredentials contains path information for a certificate and key to be used for TLS
type TLSCredentials struct {
	// CertFile is the file path containing the TLS certificate.
	CertFile string `mapstructure:"cert-file"`

	// KeyFile is the file path containing the TLS key.
	KeyFile string `mapstructure:"key-file"`
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
