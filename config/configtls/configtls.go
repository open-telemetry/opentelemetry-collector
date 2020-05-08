// Copyright 2020, OpenTelemetry Authors
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
)

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

// LoadTLSConfig loads TLS certificates and returns a tls.Config.
func (c TLSConfig) LoadTLSConfig() (*tls.Config, error) {
	certPool, err := c.loadCertPool()
	if err != nil {
		return nil, fmt.Errorf("failed to load CA CertPool: %w", err)
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

var systemCertPool = x509.SystemCertPool // to allow overriding in unit test

func (c TLSConfig) loadCertPool() (*x509.CertPool, error) {
	if len(c.CaCert) == 0 { // no truststore given, use SystemCertPool
		certPool, err := systemCertPool()
		if err != nil {
			return nil, fmt.Errorf("failed to load SystemCertPool: %w", err)
		}
		return certPool, nil
	}
	// setup user specified truststore
	return c.loadCert(c.CaCert)
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
