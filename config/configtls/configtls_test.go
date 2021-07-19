// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package configtls

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOptionsToConfig(t *testing.T) {
	tests := []struct {
		name        string
		options     TLSSetting
		expectError string
	}{
		{
			name:    "should load system CA",
			options: TLSSetting{CAFile: ""},
		},
		{
			name:    "should load custom CA",
			options: TLSSetting{CAFile: "testdata/testCA.pem"},
		},
		{
			name:        "should fail with invalid CA file path",
			options:     TLSSetting{CAFile: "testdata/not/valid"},
			expectError: "failed to load CA",
		},
		{
			name:        "should fail with invalid CA file content",
			options:     TLSSetting{CAFile: "testdata/testCA-bad.txt"},
			expectError: "failed to parse CA",
		},
		{
			name: "should load valid TLS  settings",
			options: TLSSetting{
				CAFile:   "testdata/testCA.pem",
				CertFile: "testdata/test-cert.pem",
				KeyFile:  "testdata/test-key.pem",
			},
		},
		{
			name: "should fail with missing TLS KeyFile",
			options: TLSSetting{
				CAFile:   "testdata/testCA.pem",
				CertFile: "testdata/test-cert.pem",
			},
			expectError: "both certificate and key must be supplied",
		},
		{
			name: "should fail with invalid TLS KeyFile",
			options: TLSSetting{
				CAFile:   "testdata/testCA.pem",
				CertFile: "testdata/test-cert.pem",
				KeyFile:  "testdata/not/valid",
			},
			expectError: "failed to load TLS cert and key",
		},
		{
			name: "should fail with missing TLS Cert",
			options: TLSSetting{
				CAFile:  "testdata/testCA.pem",
				KeyFile: "testdata/test-key.pem",
			},
			expectError: "both certificate and key must be supplied",
		},
		{
			name: "should fail with invalid TLS Cert",
			options: TLSSetting{
				CAFile:   "testdata/testCA.pem",
				CertFile: "testdata/not/valid",
				KeyFile:  "testdata/test-key.pem",
			},
			expectError: "failed to load TLS cert and key",
		},
		{
			name: "should fail with invalid TLS CA",
			options: TLSSetting{
				CAFile: "testdata/not/valid",
			},
			expectError: "failed to load CA",
		},
		{
			name: "should fail with invalid CA pool",
			options: TLSSetting{
				CAFile: "testdata/testCA-bad.txt",
			},
			expectError: "failed to parse CA",
		},
		{
			name: "should pass with valid CA pool",
			options: TLSSetting{
				CAFile: "testdata/testCA.pem",
			},
		},
		{
			name: "should pass with valid min and max version",
			options: TLSSetting{
				MinVersion: "1.1",
				MaxVersion: "1.2",
			},
		},
		{
			name: "should pass with invalid min",
			options: TLSSetting{
				MinVersion: "1.7",
			},
			expectError: "invalid TLS min_",
		},
		{
			name: "should pass with invalid max",
			options: TLSSetting{
				MaxVersion: "1.7",
			},
			expectError: "invalid TLS max_",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg, err := test.options.loadTLSConfig()
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

func TestLoadTLSClientConfigError(t *testing.T) {
	tlsSetting := TLSClientSetting{
		TLSSetting: TLSSetting{
			CertFile: "doesnt/exist",
			KeyFile:  "doesnt/exist",
		},
	}
	_, err := tlsSetting.LoadTLSConfig()
	assert.Error(t, err)
}

func TestLoadTLSClientConfig(t *testing.T) {
	tlsSetting := TLSClientSetting{
		Insecure: true,
	}
	tlsCfg, err := tlsSetting.LoadTLSConfig()
	assert.NoError(t, err)
	assert.Nil(t, tlsCfg)

	tlsSetting = TLSClientSetting{}
	tlsCfg, err = tlsSetting.LoadTLSConfig()
	assert.NoError(t, err)
	assert.NotNil(t, tlsCfg)

	tlsSetting = TLSClientSetting{
		InsecureSkipVerify: true,
	}
	tlsCfg, err = tlsSetting.LoadTLSConfig()
	assert.NoError(t, err)
	assert.NotNil(t, tlsCfg)
	assert.True(t, tlsCfg.InsecureSkipVerify)
}

func TestLoadTLSServerConfigError(t *testing.T) {
	tlsSetting := TLSServerSetting{
		TLSSetting: TLSSetting{
			CertFile: "doesnt/exist",
			KeyFile:  "doesnt/exist",
		},
	}
	_, err := tlsSetting.LoadTLSConfig()
	assert.Error(t, err)

	tlsSetting = TLSServerSetting{
		ClientCAFile: "doesnt/exist",
	}
	_, err = tlsSetting.LoadTLSConfig()
	assert.Error(t, err)
}

func TestLoadTLSServerConfig(t *testing.T) {
	tlsSetting := TLSServerSetting{}
	tlsCfg, err := tlsSetting.LoadTLSConfig()
	assert.NoError(t, err)
	assert.NotNil(t, tlsCfg)
}
