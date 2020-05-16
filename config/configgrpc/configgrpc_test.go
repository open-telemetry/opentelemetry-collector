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

package configgrpc

import (
	"crypto/x509"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBasicGrpcSettings(t *testing.T) {

	_, err := GrpcSettingsToDialOptions(GRPCSettings{
		Headers:     nil,
		Endpoint:    "",
		Compression: "",
		TLSConfig: TLSConfig{
			CaCert:             "",
			UseSecure:          false,
			ServerNameOverride: "",
		},
		KeepaliveParameters: nil,
	})

	assert.NoError(t, err)
}

func TestInvalidPemFile(t *testing.T) {
	tests := []struct {
		settings GRPCSettings
		err      string
	}{
		{
			err: "open /doesnt/exist: no such file or directory",
			settings: GRPCSettings{
				Headers:     nil,
				Endpoint:    "",
				Compression: "",
				TLSConfig: TLSConfig{
					CaCert:             "/doesnt/exist",
					UseSecure:          false,
					ServerNameOverride: "",
				},
				KeepaliveParameters: nil,
			},
		},
		{
			err: "failed to load TLS config: failed to load CA CertPool: failed to load CA /doesnt/exist: open /doesnt/exist: no such file or directory",
			settings: GRPCSettings{
				Headers:     nil,
				Endpoint:    "",
				Compression: "",
				TLSConfig: TLSConfig{
					CaCert:             "/doesnt/exist",
					UseSecure:          true,
					ServerNameOverride: "",
				},
				KeepaliveParameters: nil,
			},
		},
		{
			err: "failed to load TLS config: for client auth via TLS, either both client certificate and key must be supplied, or neither",
			settings: GRPCSettings{
				Headers:     nil,
				Endpoint:    "",
				Compression: "",
				TLSConfig: TLSConfig{
					ClientCert:         "/doesnt/exist",
					UseSecure:          true,
					ServerNameOverride: "",
				},
				KeepaliveParameters: nil,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.err, func(t *testing.T) {
			_, err := GrpcSettingsToDialOptions(test.settings)
			assert.EqualError(t, err, test.err)
		})
	}
}

func TestUseSecure(t *testing.T) {
	dialOpts, err := GrpcSettingsToDialOptions(GRPCSettings{
		Headers:     nil,
		Endpoint:    "",
		Compression: "",
		TLSConfig: TLSConfig{
			CaCert:             "",
			UseSecure:          true,
			ServerNameOverride: "",
		},
		KeepaliveParameters: nil,
	})

	assert.NoError(t, err)
	assert.Equal(t, len(dialOpts), 1)
}

func TestOptionsToConfig(t *testing.T) {
	tests := []struct {
		name        string
		options     TLSConfig
		fakeSysPool bool
		expectError string
	}{
		{
			name:    "should load system CA",
			options: TLSConfig{CaCert: ""},
		},
		{
			name:        "should fail with fake system CA",
			fakeSysPool: true,
			options:     TLSConfig{CaCert: ""},
			expectError: "fake system pool",
		},
		{
			name:    "should load custom CA",
			options: TLSConfig{CaCert: "testdata/testCA.pem"},
		},
		{
			name:        "should fail with invalid CA file path",
			options:     TLSConfig{CaCert: "testdata/not/valid"},
			expectError: "failed to load CA",
		},
		{
			name:        "should fail with invalid CA file content",
			options:     TLSConfig{CaCert: "testdata/testCA-bad.txt"},
			expectError: "failed to parse CA",
		},
		{
			name: "should load valid TLS Client settings",
			options: TLSConfig{
				CaCert:     "testdata/testCA.pem",
				ClientCert: "testdata/test-cert.pem",
				ClientKey:  "testdata/test-key.pem",
			},
		},
		{
			name: "should fail with missing TLS Client Key",
			options: TLSConfig{
				CaCert:     "testdata/testCA.pem",
				ClientCert: "testdata/test-cert.pem",
			},
			expectError: "both client certificate and key must be supplied",
		},
		{
			name: "should fail with invalid TLS Client Key",
			options: TLSConfig{
				CaCert:     "testdata/testCA.pem",
				ClientCert: "testdata/test-cert.pem",
				ClientKey:  "testdata/not/valid",
			},
			expectError: "failed to load server TLS cert and key",
		},
		{
			name: "should fail with missing TLS Client Cert",
			options: TLSConfig{
				CaCert:    "testdata/testCA.pem",
				ClientKey: "testdata/test-key.pem",
			},
			expectError: "both client certificate and key must be supplied",
		},
		{
			name: "should fail with invalid TLS Client Cert",
			options: TLSConfig{
				CaCert:     "testdata/testCA.pem",
				ClientCert: "testdata/not/valid",
				ClientKey:  "testdata/test-key.pem",
			},
			expectError: "failed to load server TLS cert and key",
		},
		{
			name: "should fail with invalid TLS Client CA",
			options: TLSConfig{
				CaCert: "testdata/not/valid",
			},
			expectError: "failed to load CA",
		},
		{
			name: "should fail with invalid Client CA pool",
			options: TLSConfig{
				CaCert: "testdata/testCA-bad.txt",
			},
			expectError: "failed to parse CA",
		},
		{
			name: "should pass with valid Client CA pool",
			options: TLSConfig{
				CaCert: "testdata/testCA.pem",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.fakeSysPool {
				saveSystemCertPool := systemCertPool
				systemCertPool = func() (*x509.CertPool, error) {
					return nil, fmt.Errorf("fake system pool")
				}
				defer func() {
					systemCertPool = saveSystemCertPool
				}()
			}
			cfg, err := test.options.LoadTLSConfig()
			if test.expectError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.expectError)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, cfg)
			}
		})
	}
}

func TestGetGRPCCompressionKey(t *testing.T) {
	if GetGRPCCompressionKey("gzip") != CompressionGzip {
		t.Error("gzip is marked as supported but returned unsupported")
	}

	if GetGRPCCompressionKey("Gzip") != CompressionGzip {
		t.Error("Capitalization of CompressionGzip should not matter")
	}

	if GetGRPCCompressionKey("badType") != CompressionUnsupported {
		t.Error("badType is not supported but was returned as supported")
	}
}
