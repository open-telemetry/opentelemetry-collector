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
	"fmt"
	"io/ioutil"
	"path/filepath"
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
			name:    "should load custom CA file",
			options: TLSSetting{CAFile: "testdata/testCA.pem"},
		},
		{
			name:    "should load custom CA PEM",
			options: TLSSetting{CAPem: readFilePanics("testdata/testCA.pem")},
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
			name:        "should fail with invalid CA PEM",
			options:     TLSSetting{CAPem: readFilePanics("testdata/testCA-bad.txt")},
			expectError: "failed to parse CA",
		},
		{
			name: "should fail CA file and PEM both provided",
			options: TLSSetting{
				CAFile: "testdata/testCA.pem",
				CAPem:  readFilePanics("testdata/testCA.pem"),
			},
			expectError: "failed to load CA CertPool: CA File and PEM cannot both be provided",
		},
		{
			name: "should fail Cert file and PEM both provided",
			options: TLSSetting{
				CertFile: "testdata/test-cert.pem",
				CertPem:  readFilePanics("testdata/test-cert.pem"),
			},
			expectError: "for auth via TLS, certificate file and PEM cannot both be provided",
		},
		{
			name: "should fail Cert file and Key PEM provided",
			options: TLSSetting{
				CertFile: "testdata/test-cert.pem",
				KeyPem:   readFilePanics("testdata/test-key.pem"),
			},
			expectError: "failed to load TLS cert file and key PEM: both must be provided as a file or both as a PEM",
		},
		{
			name: "should fail Cert PEM and Key File provided",
			options: TLSSetting{
				CertPem: readFilePanics("testdata/test-cert.pem"),
				KeyFile: "testdata/test-key.pem",
			},
			expectError: "failed to load TLS cert PEM and key file: both must be provided as a file or both as a PEM",
		},
		{
			name: "should fail Key file and PEM both provided",
			options: TLSSetting{
				KeyFile: "testdata/testCA.pem",
				KeyPem:  readFilePanics("testdata/test-key.pem"),
			},
			expectError: "for auth via TLS, key file and PEM cannot both be provided",
		},
		{
			name: "should load valid TLS settings with files",
			options: TLSSetting{
				CAFile:   "testdata/testCA.pem",
				CertFile: "testdata/test-cert.pem",
				KeyFile:  "testdata/test-key.pem",
			},
		},
		{
			name: "should fail to load valid TLS settings with bad Cert PEM",
			options: TLSSetting{
				CAPem:   readFilePanics("testdata/testCA.pem"),
				CertPem: readFilePanics("testdata/testCert-bad.txt"),
				KeyPem:  readFilePanics("testdata/test-key.pem"),
			},
			expectError: "failed to load TLS cert and key PEMs",
		},
		{
			name: "should fail to load valid TLS settings with bad Key PEM",
			options: TLSSetting{
				CAPem:   readFilePanics("testdata/testCA.pem"),
				CertPem: readFilePanics("testdata/test-cert.pem"),
				KeyPem:  readFilePanics("testdata/testKey-bad.txt"),
			},
			expectError: "failed to load TLS cert and key PEMs",
		},
		{
			name: "should load valid TLS settings with PEMs",
			options: TLSSetting{
				CAPem:   readFilePanics("testdata/testCA.pem"),
				CertPem: readFilePanics("testdata/test-cert.pem"),
				KeyPem:  readFilePanics("testdata/test-key.pem"),
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
			name: "should fail with missing TLS KeyPem",
			options: TLSSetting{
				CAPem:   readFilePanics("testdata/testCA.pem"),
				CertPem: readFilePanics("testdata/test-cert.pem"),
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
			name: "should fail with missing TLS Cert file",
			options: TLSSetting{
				CAFile:  "testdata/testCA.pem",
				KeyFile: "testdata/test-key.pem",
			},
			expectError: "both certificate and key must be supplied",
		},
		{
			name: "should fail with missing TLS Cert PEM",
			options: TLSSetting{
				CAPem:  readFilePanics("testdata/testCA.pem"),
				KeyPem: readFilePanics("testdata/test-key.pem"),
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
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
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

func TestTLSSetting_LoadgRPCTLSServerCredentialsError(t *testing.T) {
	tlsSetting := TLSSetting{
		CertFile: "doesnt/exist",
		KeyFile:  "doesnt/exist",
	}
	_, err := tlsSetting.LoadgRPCTLSServerCredentials()
	assert.Error(t, err)
}

func readFilePanics(filePath string) []byte {
	fileContents, err := ioutil.ReadFile(filepath.Clean(filePath))
	if err != nil {
		panic(fmt.Sprintf("failed to read file %s: %v", filePath, err))
	}

	return fileContents
}
