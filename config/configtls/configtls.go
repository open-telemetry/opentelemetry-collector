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

package configtls

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// TLSSetting exposes the common client and server TLS configurations.
// Note: Since there isn't anything specific to a server connection. Components
// with server connections should use TLSSetting.
type TLSSetting struct {
	// Path to the CA cert. For a client this verifies the server certificate.
	// For a server this verifies client certificates. If empty uses system root CA.
	// (optional)
	CAFile string `mapstructure:"ca_file"`

	// PEM encoded cert content. (optional)
	CAPem []byte `mapstructure:"ca_pem"`

	// Path to the TLS cert to use for TLS required connections. (optional)
	CertFile string `mapstructure:"cert_file"`

	// PEM encoded TLS cert to use for TLS required connections. (optional)
	CertPem []byte `mapstructure:"cert_pem"`

	// Path to the TLS key to use for TLS required connections. (optional)
	KeyFile string `mapstructure:"key_file"`

	// PEM encoded TLS key to use for TLS required connections. (optional)
	KeyPem []byte `mapstructure:"key_pem"`
}

// TLSClientSetting contains TLS configurations that are specific to client
// connections in addition to the common configurations. This should be used by
// components configuring TLS client connections.
type TLSClientSetting struct {
	TLSSetting `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct

	// These are config options specific to client connections.

	// In gRPC when set to true, this is used to disable the client transport security.
	// See https://godoc.org/google.golang.org/grpc#WithInsecure.
	// In HTTP, this disables verifying the server's certificate chain and host name
	// (InsecureSkipVerify in the tls Config). Please refer to
	// https://godoc.org/crypto/tls#Config for more information.
	// (optional, default false)
	// TODO(ccaraman): With further research InsecureSkipVerify is a valid option
	// for gRPC connections. Add that ability to the TLSClientSettings in a subsequent
	// pr.
	Insecure bool `mapstructure:"insecure"`
	// ServerName requested by client for virtual hosting.
	// This sets the ServerName in the TLSConfig. Please refer to
	// https://godoc.org/crypto/tls#Config for more information. (optional)
	ServerName string `mapstructure:"server_name_override"`
}

// LoadTLSConfig loads TLS certificates and returns a tls.Config.
// This will set the RootCAs and Certificates of a tls.Config.
func (c TLSSetting) LoadTLSConfig() (*tls.Config, error) {
	// There is no need to load the System Certs for RootCAs because
	// if the value is nil, it will default to checking against th System Certs.
	certPool, err := c.loadCertPool(c.CAFile, c.CAPem)
	if err != nil {
		return nil, err
	}

	if len(c.CertFile) != 0 && len(c.CertPem) != 0 {
		return nil, fmt.Errorf("for auth via TLS, certificate file and PEM cannot both be provided")
	}

	if len(c.KeyFile) != 0 && len(c.KeyPem) != 0 {
		return nil, fmt.Errorf("for auth via TLS, key file and PEM cannot both be provided")
	}

	hasCert := len(c.CertFile)+len(c.CertPem) > 0
	hasKey := len(c.KeyFile)+len(c.KeyPem) > 0
	if hasCert != hasKey {
		return nil, fmt.Errorf("for auth via TLS, either both certificate and key must be supplied, or neither")
	}

	var certificates []tls.Certificate
	if hasCert && hasKey {
		var tlsCert tls.Certificate
		if len(c.CertFile) > 0 {
			if len(c.KeyFile) == 0 {
				return nil, fmt.Errorf("failed to load TLS cert file and key PEM: both must be provided as a file or both as a PEM")
			}
			tlsCert, err = tls.LoadX509KeyPair(filepath.Clean(c.CertFile), filepath.Clean(c.KeyFile))
			if err != nil {
				return nil, fmt.Errorf("failed to load TLS cert and key files: %w", err)
			}
		} else {
			if len(c.KeyFile) > 0 {
				return nil, fmt.Errorf("failed to load TLS cert PEM and key file: both must be provided as a file or both as a PEM")
			}
			tlsCert, err = tls.X509KeyPair(c.CertPem, []byte(c.KeyPem))
			if err != nil {
				return nil, fmt.Errorf("failed to load TLS cert and key PEMs: %w", err)
			}
		}

		certificates = append(certificates, tlsCert)
	}

	return &tls.Config{
		RootCAs:      certPool,
		Certificates: certificates,
	}, nil
}

func (c TLSSetting) loadCertPool(caPath string, caPEM []byte) (*x509.CertPool, error) {
	var err error
	var certPool *x509.CertPool

	if len(c.CAFile) != 0 && len(c.CAPem) != 0 {
		return nil, fmt.Errorf("failed to load CA CertPool: CA File and PEM cannot both be provided")
	} else if len(c.CAFile) != 0 {
		// setup user specified truststore
		certPool, err = c.loadCertFile(c.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load CA CertPool File: %w", err)
		}
	} else if len(c.CAPem) != 0 {
		certPool, err = c.loadCertPem([]byte(c.CAPem))
		if err != nil {
			return nil, fmt.Errorf("failed to load CA CertPool PEM: %w", err)
		}
	}

	return certPool, nil
}

func (c TLSSetting) loadCertFile(caPath string) (*x509.CertPool, error) {
	caPEM, err := ioutil.ReadFile(filepath.Clean(caPath))
	if err != nil {
		return nil, fmt.Errorf("failed to load CA %s: %w", caPath, err)
	}

	return c.loadCertPem(caPEM)
}

func (c TLSSetting) loadCertPem(caPEM []byte) (*x509.CertPool, error) {
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("failed to parse CA %s", caPEM)
	}
	return certPool, nil
}

func (c TLSClientSetting) LoadgRPCTLSClientCredentials() (grpc.DialOption, error) {
	if c.Insecure && c.CAFile == "" {
		return grpc.WithInsecure(), nil
	}

	tlsConf, err := c.TLSSetting.LoadTLSConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load TLS config: %w", err)
	}
	tlsConf.ServerName = c.ServerName
	creds := credentials.NewTLS(tlsConf)
	return grpc.WithTransportCredentials(creds), nil
}

func (c TLSSetting) LoadgRPCTLSServerCredentials() (grpc.ServerOption, error) {
	tlsConf, err := c.LoadTLSConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load TLS config: %w", err)
	}
	creds := credentials.NewTLS(tlsConf)
	return grpc.Creds(creds), nil
}
