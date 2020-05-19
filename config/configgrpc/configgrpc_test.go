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
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/config/configtls"
)

func TestBasicGrpcSettings(t *testing.T) {

	_, err := GrpcSettingsToDialOptions(GRPCClientSettings{
		Headers:             nil,
		Endpoint:            "",
		Compression:         "",
		KeepaliveParameters: nil,
	})

	assert.NoError(t, err)
}

func TestInvalidPemFile(t *testing.T) {
	tests := []struct {
		settings GRPCClientSettings
		err      string
	}{
		{
			err: "open /doesnt/exist: no such file or directory",
			settings: GRPCClientSettings{
				Headers:     nil,
				Endpoint:    "",
				Compression: "",
				TLSConfig: configtls.TLSClientConfig{
					TLSConfig: configtls.TLSConfig{
						CAFile: "/doesnt/exist",
					},
					UseInsecure: true,
					ServerName:  "",
				},
				KeepaliveParameters: nil,
			},
		},
		{
			err: "failed to load TLS config: failed to load CA CertPool: failed to load CA /doesnt/exist: open /doesnt/exist: no such file or directory",
			settings: GRPCClientSettings{
				Headers:     nil,
				Endpoint:    "",
				Compression: "",
				TLSConfig: configtls.TLSClientConfig{
					TLSConfig: configtls.TLSConfig{
						CAFile: "/doesnt/exist",
					},
					UseInsecure: false,
					ServerName:  "",
				},
				KeepaliveParameters: nil,
			},
		},
		{
			err: "failed to load TLS config: for auth via TLS, either both certificate and key must be supplied, or neither",
			settings: GRPCClientSettings{
				Headers:     nil,
				Endpoint:    "",
				Compression: "",
				TLSConfig: configtls.TLSClientConfig{
					TLSConfig: configtls.TLSConfig{
						CertFile: "/doesnt/exist",
					},
					UseInsecure: false,
					ServerName:  "",
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
	dialOpts, err := GrpcSettingsToDialOptions(GRPCClientSettings{
		Headers:             nil,
		Endpoint:            "",
		Compression:         "",
		TLSConfig:           configtls.TLSClientConfig{},
		KeepaliveParameters: nil,
	})

	assert.NoError(t, err)
	assert.Equal(t, len(dialOpts), 1)
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
