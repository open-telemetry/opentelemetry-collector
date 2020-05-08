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

	"github.com/open-telemetry/opentelemetry-collector/config/configtls"
)

func TestBasicGrpcSettings(t *testing.T) {

	_, err := GrpcSettingsToDialOptions(GRPCSettings{
		Headers:     nil,
		Endpoint:    "",
		Compression: "",
		TLSConfig: configtls.TLSConfig{
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
				TLSConfig: configtls.TLSConfig{
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
				TLSConfig: configtls.TLSConfig{
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
				TLSConfig: configtls.TLSConfig{
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
		TLSConfig: configtls.TLSConfig{
			CaCert:             "",
			UseSecure:          true,
			ServerNameOverride: "",
		},
		KeepaliveParameters: nil,
	})

	assert.NoError(t, err)
	assert.Equal(t, len(dialOpts), 1)
}
