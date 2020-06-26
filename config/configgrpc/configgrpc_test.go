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

package configgrpc

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/config/configtls"
)

func TestBasicGrpcSettings(t *testing.T) {
	gcs := &GRPCClientSettings{
		Headers:     nil,
		Endpoint:    "",
		Compression: "",
		Keepalive:   nil,
	}
	_, err := gcs.ToDialOptions()

	assert.NoError(t, err)
}

func TestGRPCClientSettingsError(t *testing.T) {
	tests := []struct {
		settings GRPCClientSettings
		err      string
	}{
		{
			err: "^failed to load TLS config: failed to load CA CertPool: failed to load CA /doesnt/exist:",
			settings: GRPCClientSettings{
				Headers:     nil,
				Endpoint:    "",
				Compression: "",
				TLSSetting: configtls.TLSClientSetting{
					TLSSetting: configtls.TLSSetting{
						CAFile: "/doesnt/exist",
					},
					Insecure:   false,
					ServerName: "",
				},
				Keepalive: nil,
			},
		},
		{
			err: "^failed to load TLS config: for auth via TLS, either both certificate and key must be supplied, or neither",
			settings: GRPCClientSettings{
				Headers:     nil,
				Endpoint:    "",
				Compression: "",
				TLSSetting: configtls.TLSClientSetting{
					TLSSetting: configtls.TLSSetting{
						CertFile: "/doesnt/exist",
					},
					Insecure:   false,
					ServerName: "",
				},
				Keepalive: nil,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.err, func(t *testing.T) {
			_, err := test.settings.ToDialOptions()
			assert.Regexp(t, test.err, err)
		})
	}
}

func TestUseSecure(t *testing.T) {
	gcs := &GRPCClientSettings{
		Headers:     nil,
		Endpoint:    "",
		Compression: "",
		TLSSetting:  configtls.TLSClientSetting{},
		Keepalive:   nil,
	}
	dialOpts, err := gcs.ToDialOptions()
	assert.NoError(t, err)
	assert.Equal(t, len(dialOpts), 1)
}

func TestGRPCServerSettingsError(t *testing.T) {
	tests := []struct {
		settings GRPCServerSettings
		err      string
	}{
		{
			err: "^failed to load TLS config: failed to load CA CertPool: failed to load CA /doesnt/exist:",
			settings: GRPCServerSettings{
				Endpoint: "",
				TLSCredentials: &configtls.TLSServerSetting{
					TLSSetting: configtls.TLSSetting{
						CAFile: "/doesnt/exist",
					},
				},
				Keepalive: nil,
			},
		},
		{
			err: "^failed to load TLS config: for auth via TLS, either both certificate and key must be supplied, or neither",
			settings: GRPCServerSettings{
				Endpoint: "",
				TLSCredentials: &configtls.TLSServerSetting{
					TLSSetting: configtls.TLSSetting{
						CertFile: "/doesnt/exist",
					},
				},
				Keepalive: nil,
			},
		},
		{
			err: "^failed to load TLS config: failed to load client CA CertPool: failed to load CA /doesnt/exist:",
			settings: GRPCServerSettings{
				Endpoint: "",
				TLSCredentials: &configtls.TLSServerSetting{
					ClientCAFile: "/doesnt/exist",
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.err, func(t *testing.T) {
			_, err := test.settings.ToServerOption()
			assert.Regexp(t, test.err, err)
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
