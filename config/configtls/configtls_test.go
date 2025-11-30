// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configtls

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/confmap/xconfmap"
)

func TestNewDefaultConfig(t *testing.T) {
	expectedConfig := Config{}
	config := NewDefaultConfig()
	require.Equal(t, expectedConfig, config)
}

func TestNewDefaultClientConfig(t *testing.T) {
	expectedConfig := ClientConfig{
		Config: NewDefaultConfig(),
	}
	config := NewDefaultClientConfig()
	require.Equal(t, expectedConfig, config)
}

func TestNewDefaultServerConfig(t *testing.T) {
	expectedConfig := ServerConfig{
		Config: NewDefaultConfig(),
	}
	config := NewDefaultServerConfig()
	require.Equal(t, expectedConfig, config)
}

func TestOptionsToConfig(t *testing.T) {
	tests := []struct {
		name        string
		options     Config
		expectError string
	}{
		{
			name:    "should load system CA",
			options: Config{CAFile: ""},
		},
		{
			name:    "should load custom CA",
			options: Config{CAFile: filepath.Join("testdata", "ca-1.crt")},
		},
		{
			name:    "should load system CA and custom CA",
			options: Config{IncludeSystemCACertsPool: true, CAFile: filepath.Join("testdata", "ca-1.crt")},
		},
		{
			name:        "should fail with invalid CA file path",
			options:     Config{CAFile: filepath.Join("testdata", "not/valid")},
			expectError: "failed to load CA",
		},
		{
			name:        "should fail with invalid CA file content",
			options:     Config{CAFile: filepath.Join("testdata", "testCA-bad.txt")},
			expectError: "failed to parse cert",
		},
		{
			name: "should load valid TLS  settings",
			options: Config{
				CAFile:   filepath.Join("testdata", "ca-1.crt"),
				CertFile: filepath.Join("testdata", "server-1.crt"),
				KeyFile:  filepath.Join("testdata", "server-1.key"),
			},
		},
		{
			name: "should fail with missing TLS KeyFile",
			options: Config{
				CAFile:   filepath.Join("testdata", "ca-1.crt"),
				CertFile: filepath.Join("testdata", "server-1.crt"),
			},
			expectError: "provide both certificate and key, or neither",
		},
		{
			name: "should fail with invalid TLS KeyFile",
			options: Config{
				CAFile:   filepath.Join("testdata", "ca-1.crt"),
				CertFile: filepath.Join("testdata", "server-1.crt"),
				KeyFile:  filepath.Join("testdata", "not/valid"),
			},
			expectError: "failed to load TLS cert and key",
		},
		{
			name: "should fail with missing TLS Cert",
			options: Config{
				CAFile:  filepath.Join("testdata", "ca-1.crt"),
				KeyFile: filepath.Join("testdata", "server-1.key"),
			},
			expectError: "provide both certificate and key, or neither",
		},
		{
			name: "should fail with invalid TLS Cert",
			options: Config{
				CAFile:   filepath.Join("testdata", "ca-1.crt"),
				CertFile: filepath.Join("testdata", "not/valid"),
				KeyFile:  filepath.Join("testdata", "server-1.key"),
			},
			expectError: "failed to load TLS cert and key",
		},
		{
			name: "should fail with invalid TLS CA",
			options: Config{
				CAFile: filepath.Join("testdata", "not/valid"),
			},
			expectError: "failed to load CA",
		},
		{
			name: "should fail with invalid CA pool",
			options: Config{
				CAFile: filepath.Join("testdata", "testCA-bad.txt"),
			},
			expectError: "failed to parse cert",
		},
		{
			name: "should pass with valid CA pool",
			options: Config{
				CAFile: filepath.Join("testdata", "ca-1.crt"),
			},
		},
		{
			name: "should pass with valid min and max version",
			options: Config{
				MinVersion: "1.1",
				MaxVersion: "1.2",
			},
		},
		{
			name: "should pass with invalid min",
			options: Config{
				MinVersion: "1.7",
			},
			expectError: "invalid TLS min_",
		},
		{
			name: "should pass with invalid max",
			options: Config{
				MaxVersion: "1.7",
			},
			expectError: "invalid TLS max_",
		},
		{
			name:    "should load custom CA PEM",
			options: Config{CAPem: readFilePanics("testdata/ca-1.crt")},
		},
		{
			name: "should load valid TLS settings with PEMs",
			options: Config{
				CAPem:   readFilePanics("testdata/ca-1.crt"),
				CertPem: readFilePanics("testdata/server-1.crt"),
				KeyPem:  readFilePanics("testdata/server-1.key"),
			},
		},
		{
			name: "mix Cert file and Key PEM provided",
			options: Config{
				CertFile: "testdata/server-1.crt",
				KeyPem:   readFilePanics("testdata/server-1.key"),
			},
		},
		{
			name: "mix Cert PEM and Key File provided",
			options: Config{
				CertPem: readFilePanics("testdata/server-1.crt"),
				KeyFile: "testdata/server-1.key",
			},
		},
		{
			name:        "should fail with invalid CA PEM",
			options:     Config{CAPem: readFilePanics("testdata/testCA-bad.txt")},
			expectError: "failed to parse cert",
		},
		{
			name: "should fail CA file and PEM both provided",
			options: Config{
				CAFile: "testdata/ca-1.crt",
				CAPem:  readFilePanics("testdata/ca-1.crt"),
			},
			expectError: "provide either a CA file or the PEM-encoded string, but not both",
		},
		{
			name: "should fail Cert file and PEM both provided",
			options: Config{
				CertFile: "testdata/server-1.crt",
				CertPem:  readFilePanics("testdata/server-1.crt"),
				KeyFile:  "testdata/server-1.key",
			},
			expectError: "provide either a certificate or the PEM-encoded string, but not both",
		},
		{
			name: "should fail Key file and PEM both provided",
			options: Config{
				CertFile: "testdata/server-1.crt",
				KeyFile:  "testdata/ca-1.crt",
				KeyPem:   readFilePanics("testdata/server-1.key"),
			},
			expectError: "provide either a key or the PEM-encoded string, but not both",
		},
		{
			name: "should fail to load valid TLS settings with bad Cert PEM",
			options: Config{
				CAPem:   readFilePanics("testdata/ca-1.crt"),
				CertPem: readFilePanics("testdata/testCA-bad.txt"),
				KeyPem:  readFilePanics("testdata/server-1.key"),
			},
			expectError: "failed to load TLS cert and key PEMs",
		},
		{
			name: "should fail to load valid TLS settings with bad Key PEM",
			options: Config{
				CAPem:   readFilePanics("testdata/ca-1.crt"),
				CertPem: readFilePanics("testdata/server-1.crt"),
				KeyPem:  readFilePanics("testdata/testCA-bad.txt"),
			},
			expectError: "failed to load TLS cert and key PEMs",
		},
		{
			name: "should fail with missing TLS KeyPem",
			options: Config{
				CAPem:   readFilePanics("testdata/ca-1.crt"),
				CertPem: readFilePanics("testdata/server-1.crt"),
			},
			expectError: "provide both certificate and key, or neither",
		},
		{
			name: "should fail with missing TLS Cert PEM",
			options: Config{
				CAPem:  readFilePanics("testdata/ca-1.crt"),
				KeyPem: readFilePanics("testdata/server-1.key"),
			},
			expectError: "provide both certificate and key, or neither",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg, err := test.options.loadTLSConfig()
			if test.expectError != "" {
				assert.ErrorContains(t, err, test.expectError)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, cfg)
			}
		})
	}
}

func readFilePanics(filePath string) configopaque.String {
	fileContents, err := os.ReadFile(filepath.Clean(filePath))
	if err != nil {
		panic(fmt.Sprintf("failed to read file %s: %v", filePath, err))
	}

	return configopaque.String(fileContents)
}

func TestLoadTLSClientConfigError(t *testing.T) {
	tlsSetting := ClientConfig{
		Config: Config{
			CertFile: "doesnt/exist",
			KeyFile:  "doesnt/exist",
		},
	}
	_, err := tlsSetting.LoadTLSConfig(context.Background())
	assert.Error(t, err)
}

func TestLoadTLSClientConfig(t *testing.T) {
	tlsSetting := ClientConfig{
		Insecure: true,
	}
	tlsCfg, err := tlsSetting.LoadTLSConfig(context.Background())
	require.NoError(t, err)
	assert.Nil(t, tlsCfg)

	tlsSetting = ClientConfig{}
	tlsCfg, err = tlsSetting.LoadTLSConfig(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, tlsCfg)

	tlsSetting = ClientConfig{
		InsecureSkipVerify: true,
	}
	tlsCfg, err = tlsSetting.LoadTLSConfig(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, tlsCfg)
	assert.True(t, tlsCfg.InsecureSkipVerify)
}

func TestLoadTLSServerConfigError(t *testing.T) {
	tlsSetting := ServerConfig{
		Config: Config{
			CertFile: "doesnt/exist",
			KeyFile:  "doesnt/exist",
		},
	}
	_, err := tlsSetting.LoadTLSConfig(context.Background())
	require.Error(t, err)

	tlsSetting = ServerConfig{
		ClientCAFile: "doesnt/exist",
	}
	_, err = tlsSetting.LoadTLSConfig(context.Background())
	assert.Error(t, err)
}

func TestLoadTLSServerConfig(t *testing.T) {
	tlsSetting := ServerConfig{}
	tlsCfg, err := tlsSetting.LoadTLSConfig(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, tlsCfg)
}

func TestLoadTLSServerConfigReload(t *testing.T) {
	tmpCaPath := createTempClientCaFile(t)

	overwriteClientCA(t, tmpCaPath, "ca-1.crt")

	tlsSetting := ServerConfig{
		ClientCAFile:       tmpCaPath,
		ReloadClientCAFile: true,
	}

	tlsCfg, err := tlsSetting.LoadTLSConfig(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, tlsCfg)

	firstClient, err := tlsCfg.GetConfigForClient(nil)
	require.NoError(t, err)

	overwriteClientCA(t, tmpCaPath, "ca-2.crt")

	assert.EventuallyWithT(t, func(t *assert.CollectT) {
		secondClient, err := tlsCfg.GetConfigForClient(nil)
		require.NoError(t, err)
		assert.NotEqual(t, firstClient.ClientCAs, secondClient.ClientCAs)
	}, 5*time.Second, 10*time.Millisecond)
}

func TestLoadTLSServerConfigFailingReload(t *testing.T) {
	tmpCaPath := createTempClientCaFile(t)

	overwriteClientCA(t, tmpCaPath, "ca-1.crt")

	tlsSetting := ServerConfig{
		ClientCAFile:       tmpCaPath,
		ReloadClientCAFile: true,
	}

	tlsCfg, err := tlsSetting.LoadTLSConfig(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, tlsCfg)

	firstClient, err := tlsCfg.GetConfigForClient(nil)
	require.NoError(t, err)

	overwriteClientCA(t, tmpCaPath, "testCA-bad.txt")

	assert.Eventually(t, func() bool {
		_, loadError := tlsCfg.GetConfigForClient(nil)
		return loadError == nil
	}, 5*time.Second, 10*time.Millisecond)

	secondClient, err := tlsCfg.GetConfigForClient(nil)
	require.NoError(t, err)

	assert.Equal(t, firstClient.ClientCAs, secondClient.ClientCAs)
}

func TestLoadTLSServerConfigFailingInitialLoad(t *testing.T) {
	tmpCaPath := createTempClientCaFile(t)

	overwriteClientCA(t, tmpCaPath, "testCA-bad.txt")

	tlsSetting := ServerConfig{
		ClientCAFile:       tmpCaPath,
		ReloadClientCAFile: true,
	}

	tlsCfg, err := tlsSetting.LoadTLSConfig(context.Background())
	require.Error(t, err)
	assert.Nil(t, tlsCfg)
}

func TestLoadTLSServerConfigWrongPath(t *testing.T) {
	tmpCaPath := createTempClientCaFile(t)

	tlsSetting := ServerConfig{
		ClientCAFile:       tmpCaPath + "wrong-path",
		ReloadClientCAFile: true,
	}

	tlsCfg, err := tlsSetting.LoadTLSConfig(context.Background())
	require.Error(t, err)
	assert.Nil(t, tlsCfg)
}

func TestLoadTLSServerConfigFailing(t *testing.T) {
	tmpCaPath := createTempClientCaFile(t)

	overwriteClientCA(t, tmpCaPath, "ca-1.crt")

	tlsSetting := ServerConfig{
		ClientCAFile:       tmpCaPath,
		ReloadClientCAFile: true,
	}

	tlsCfg, err := tlsSetting.LoadTLSConfig(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, tlsCfg)

	firstClient, err := tlsCfg.GetConfigForClient(nil)
	require.NoError(t, err)
	assert.NotNil(t, firstClient)

	err = os.Remove(tmpCaPath)
	require.NoError(t, err)

	firstClient, err = tlsCfg.GetConfigForClient(nil)
	require.NoError(t, err)
	assert.NotNil(t, firstClient)
}

func overwriteClientCA(t *testing.T, targetFilePath, testdataFileName string) {
	targetFile, err := os.OpenFile(filepath.Clean(targetFilePath), os.O_RDWR, 0o600)
	require.NoError(t, err)

	testdataFilePath := filepath.Join("testdata", testdataFileName)
	testdataFile, err := os.OpenFile(filepath.Clean(testdataFilePath), os.O_RDONLY, 0o200)
	require.NoError(t, err)

	_, err = io.Copy(targetFile, testdataFile)
	assert.NoError(t, err)

	assert.NoError(t, targetFile.Close())
	assert.NoError(t, testdataFile.Close())
}

func createTempClientCaFile(t *testing.T) string {
	tmpCa, err := os.CreateTemp(t.TempDir(), "ca-tmp.crt")
	require.NoError(t, err)
	tmpCaPath, err := filepath.Abs(tmpCa.Name())
	assert.NoError(t, err)
	assert.NoError(t, tmpCa.Close())
	return tmpCaPath
}

func TestEagerlyLoadCertificate(t *testing.T) {
	options := Config{
		CertFile: filepath.Join("testdata", "client-1.crt"),
		KeyFile:  filepath.Join("testdata", "client-1.key"),
	}
	cfg, err := options.loadTLSConfig()
	require.NoError(t, err)
	assert.NotNil(t, cfg)
	cert, err := cfg.GetCertificate(&tls.ClientHelloInfo{})
	require.NoError(t, err)
	assert.NotNil(t, cert)
	pCert, err := x509.ParseCertificate(cert.Certificate[0])
	require.NoError(t, err)
	assert.NotNil(t, pCert)
	assert.ElementsMatch(t, []string{"example1"}, pCert.DNSNames)
}

func TestCertificateReload(t *testing.T) {
	tests := []struct {
		name           string
		reloadInterval time.Duration
		wait           time.Duration
		cert2          string
		key2           string
		dns1           string
		dns2           string
		errText        string
	}{
		{
			name:           "Should reload the certificate after reload-interval",
			reloadInterval: 100 * time.Microsecond,
			wait:           100 * time.Microsecond,
			cert2:          "client-2.crt",
			key2:           "client-2.key",
			dns1:           "example1",
			dns2:           "example2",
		},
		{
			name:           "Should return same cert if called before reload-interval",
			reloadInterval: 100 * time.Millisecond,
			wait:           100 * time.Microsecond,
			cert2:          "client-2.crt",
			key2:           "client-2.key",
			dns1:           "example1",
			dns2:           "example1",
		},
		{
			name:           "Should always return same cert if reload-interval is 0",
			reloadInterval: 0,
			wait:           100 * time.Microsecond,
			cert2:          "client-2.crt",
			key2:           "client-2.key",
			dns1:           "example1",
			dns2:           "example1",
		},
		{
			name:           "Should return an error if reloading fails",
			reloadInterval: 100 * time.Microsecond,
			wait:           100 * time.Microsecond,
			cert2:          "testCA-bad.txt",
			key2:           "client-2.key",
			dns1:           "example1",
			errText:        "failed to load TLS cert and key: failed to load TLS cert and key PEMs: tls: failed to find any PEM data in certificate input",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Copy certs into a temp dir so we can safely modify them
			tempDir := t.TempDir()
			certFile, err := os.CreateTemp(tempDir, "cert")
			require.NoError(t, err)
			defer certFile.Close()

			keyFile, err := os.CreateTemp(tempDir, "key")
			require.NoError(t, err)
			defer keyFile.Close()

			fdc, err := os.Open(filepath.Join("testdata", "client-1.crt"))
			require.NoError(t, err)
			_, err = io.Copy(certFile, fdc)
			require.NoError(t, err)
			require.NoError(t, fdc.Close())

			fdk, err := os.Open(filepath.Join("testdata", "client-1.key"))
			require.NoError(t, err)
			_, err = io.Copy(keyFile, fdk)
			assert.NoError(t, err)
			assert.NoError(t, fdk.Close())

			options := Config{
				CertFile:       certFile.Name(),
				KeyFile:        keyFile.Name(),
				ReloadInterval: test.reloadInterval,
			}
			cfg, err := options.loadTLSConfig()
			require.NoError(t, err)
			assert.NotNil(t, cfg)

			// Assert that we loaded the original certificate
			cert, err := cfg.GetCertificate(&tls.ClientHelloInfo{})
			require.NoError(t, err)
			assert.NotNil(t, cert)
			pCert, err := x509.ParseCertificate(cert.Certificate[0])
			require.NoError(t, err)
			assert.NotNil(t, pCert)
			assert.Equal(t, test.dns1, pCert.DNSNames[0])

			// Change the certificate
			assert.NoError(t, certFile.Truncate(0))
			assert.NoError(t, keyFile.Truncate(0))
			_, err = certFile.Seek(0, 0)
			require.NoError(t, err)
			_, err = keyFile.Seek(0, 0)
			require.NoError(t, err)

			fdc2, err := os.Open(filepath.Join("testdata", test.cert2))
			require.NoError(t, err)
			_, err = io.Copy(certFile, fdc2)
			assert.NoError(t, err)
			assert.NoError(t, fdc2.Close())

			fdk2, err := os.Open(filepath.Join("testdata", test.key2))
			require.NoError(t, err)
			_, err = io.Copy(keyFile, fdk2)
			assert.NoError(t, err)
			assert.NoError(t, fdk2.Close())

			// Wait ReloadInterval to ensure a reload will happen
			time.Sleep(test.wait)

			// Assert that we loaded the new certificate
			cert, err = cfg.GetCertificate(&tls.ClientHelloInfo{})
			if test.errText == "" {
				require.NoError(t, err)
				assert.NotNil(t, cert)
				pCert, err = x509.ParseCertificate(cert.Certificate[0])
				require.NoError(t, err)
				assert.NotNil(t, pCert)
				assert.Equal(t, test.dns2, pCert.DNSNames[0])
			} else {
				assert.EqualError(t, err, test.errText)
			}
		})
	}
}

func TestMinMaxTLSVersions(t *testing.T) {
	tests := []struct {
		name          string
		minVersion    string
		maxVersion    string
		outMinVersion uint16
		outMaxVersion uint16
		errorTxt      string
	}{
		{name: `TLS Config ["", ""] to give [TLS1.2, 0]`, minVersion: "", maxVersion: "", outMinVersion: tls.VersionTLS12, outMaxVersion: 0},
		{name: `TLS Config ["", "1.3"] to give [TLS1.2, TLS1.3]`, minVersion: "", maxVersion: "1.3", outMinVersion: tls.VersionTLS12, outMaxVersion: tls.VersionTLS13},
		{name: `TLS Config ["1.2", ""] to give [TLS1.2, 0]`, minVersion: "1.2", maxVersion: "", outMinVersion: tls.VersionTLS12, outMaxVersion: 0},
		{name: `TLS Config ["1.3", "1.3"] to give [TLS1.3, TLS1.3]`, minVersion: "1.3", maxVersion: "1.3", outMinVersion: tls.VersionTLS13, outMaxVersion: tls.VersionTLS13},
		{name: `TLS Config ["1.0", "1.1"] to give [TLS1.0, TLS1.1]`, minVersion: "1.0", maxVersion: "1.1", outMinVersion: tls.VersionTLS10, outMaxVersion: tls.VersionTLS11},
		{name: `TLS Config ["asd", ""] to give [Error]`, minVersion: "asd", maxVersion: "", errorTxt: `invalid TLS min_version: unsupported TLS version: "asd"`},
		{name: `TLS Config ["", "asd"] to give [Error]`, minVersion: "", maxVersion: "asd", errorTxt: `invalid TLS max_version: unsupported TLS version: "asd"`},
		{name: `TLS Config ["0.4", ""] to give [Error]`, minVersion: "0.4", maxVersion: "", errorTxt: `invalid TLS min_version: unsupported TLS version: "0.4"`},

		// Allowing this, however, expecting downstream TLS handshake will throw an error
		{name: `TLS Config ["1.2", "1.1"] to give [TLS1.2, TLS1.1]`, minVersion: "1.2", maxVersion: "1.1", outMinVersion: tls.VersionTLS12, outMaxVersion: tls.VersionTLS11},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setting := Config{
				MinVersion: test.minVersion,
				MaxVersion: test.maxVersion,
			}

			config, err := setting.loadTLSConfig()

			if test.errorTxt == "" {
				assert.Equal(t, config.MinVersion, test.outMinVersion)
				assert.Equal(t, config.MaxVersion, test.outMaxVersion)
			} else {
				assert.EqualError(t, err, test.errorTxt)
			}
		})
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name      string
		tlsConfig Config
		errorTxt  string
	}{
		{name: `TLS Config ["", ""] to be valid`, tlsConfig: Config{MinVersion: "", MaxVersion: ""}},
		{name: `TLS Config ["", "1.3"] to be valid`, tlsConfig: Config{MinVersion: "", MaxVersion: "1.3"}},
		{name: `TLS Config ["1.2", ""] to be valid`, tlsConfig: Config{MinVersion: "1.2", MaxVersion: ""}},
		{name: `TLS Config ["1.3", "1.3"] to be valid`, tlsConfig: Config{MinVersion: "1.3", MaxVersion: "1.3"}},
		{name: `TLS Config ["1.0", "1.1"] to be valid`, tlsConfig: Config{MinVersion: "1.0", MaxVersion: "1.1"}},
		{name: `TLS Config ["asd", ""] to give [Error]`, tlsConfig: Config{MinVersion: "asd", MaxVersion: ""}, errorTxt: `invalid TLS min_version: unsupported TLS version: "asd"`},
		{name: `TLS Config ["", "asd"] to give [Error]`, tlsConfig: Config{MinVersion: "", MaxVersion: "asd"}, errorTxt: `invalid TLS max_version: unsupported TLS version: "asd"`},
		{name: `TLS Config ["0.4", ""] to give [Error]`, tlsConfig: Config{MinVersion: "0.4", MaxVersion: ""}, errorTxt: `invalid TLS min_version: unsupported TLS version: "0.4"`},
		{name: `TLS Config ["1.2", "1.1"] to give [Error]`, tlsConfig: Config{MinVersion: "1.2", MaxVersion: "1.1"}, errorTxt: `invalid TLS configuration: min_version cannot be greater than max_version`},
		{name: `TLS Config with both CA File and PEM`, tlsConfig: Config{CAFile: "test", CAPem: "test"}, errorTxt: `provide either a CA file or the PEM-encoded string, but not both`},
		{name: `TLS Config with cert file but no key`, tlsConfig: Config{CertFile: "cert.pem"}, errorTxt: `TLS configuration must include both certificate and key (CertFile/CertPem and KeyFile/KeyPem)`},
		{name: `TLS Config with key file but no cert`, tlsConfig: Config{KeyFile: "key.pem"}, errorTxt: `TLS configuration must include both certificate and key (CertFile/CertPem and KeyFile/KeyPem)`},
		{name: `TLS Config with cert PEM but no key`, tlsConfig: Config{CertPem: "cert-pem"}, errorTxt: `TLS configuration must include both certificate and key (CertFile/CertPem and KeyFile/KeyPem)`},
		{name: `TLS Config with key PEM but no cert`, tlsConfig: Config{KeyPem: "key-pem"}, errorTxt: `TLS configuration must include both certificate and key (CertFile/CertPem and KeyFile/KeyPem)`},
		{name: `TLS Config with both cert file and cert PEM`, tlsConfig: Config{CertFile: "cert.pem", CertPem: "cert-pem", KeyFile: "key.pem"}, errorTxt: `provide either certificate file or PEM, but not both`},
		{name: `TLS Config with both key file and key PEM`, tlsConfig: Config{CertFile: "cert.pem", KeyFile: "key.pem", KeyPem: "key-pem"}, errorTxt: `provide either key file or PEM, but not both`},
		{name: `TLS Config with cert file and key PEM`, tlsConfig: Config{CertFile: "cert.pem", KeyPem: "key-pem"}},
		{name: `TLS Config with cert PEM and key file`, tlsConfig: Config{CertPem: "cert-pem", KeyFile: "key.pem"}},
		{name: `TLS Config with valid cert and key files`, tlsConfig: Config{CertFile: "cert.pem", KeyFile: "key.pem"}},
		{name: `TLS Config with valid cert and key PEM`, tlsConfig: Config{CertPem: "cert-pem", KeyPem: "key-pem"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.tlsConfig.Validate()

			if test.errorTxt == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, test.errorTxt)
			}
		})
	}
}

func TestCipherSuites(t *testing.T) {
	tests := []struct {
		name       string
		tlsSetting Config
		wantErr    string
		result     []uint16
	}{
		{
			name:       "no suites set",
			tlsSetting: Config{},
			result:     nil,
		},
		{
			name: "one cipher suite set",
			tlsSetting: Config{
				CipherSuites: []string{"TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA"},
			},
			result: []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA},
		},
		{
			name: "invalid cipher suite set",
			tlsSetting: Config{
				CipherSuites: []string{"FOO"},
			},
			wantErr: `invalid TLS cipher suite: "FOO"`,
		},
		{
			name: "multiple invalid cipher suites set",
			tlsSetting: Config{
				CipherSuites: []string{"FOO", "BAR"},
			},
			wantErr: `invalid TLS cipher suite: "FOO"
invalid TLS cipher suite: "BAR"`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config, err := test.tlsSetting.loadTLSConfig()
			if test.wantErr != "" {
				assert.EqualError(t, err, test.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.result, config.CipherSuites)
			}
		})
	}
}

func TestSystemCertPool(t *testing.T) {
	anError := errors.New("my error")
	tests := []struct {
		name         string
		tlsConfig    Config
		wantErr      error
		systemCertFn func() (*x509.CertPool, error)
	}{
		{
			name: "not using system cert pool",
			tlsConfig: Config{
				IncludeSystemCACertsPool: false,
				CAFile:                   filepath.Join("testdata", "ca-1.crt"),
			},
			wantErr:      nil,
			systemCertFn: x509.SystemCertPool,
		},
		{
			name: "using system cert pool",
			tlsConfig: Config{
				IncludeSystemCACertsPool: true,
				CAFile:                   filepath.Join("testdata", "ca-1.crt"),
			},
			wantErr:      nil,
			systemCertFn: x509.SystemCertPool,
		},
		{
			name: "error loading system cert pool",
			tlsConfig: Config{
				IncludeSystemCACertsPool: true,
				CAFile:                   filepath.Join("testdata", "ca-1.crt"),
			},
			wantErr: anError,
			systemCertFn: func() (*x509.CertPool, error) {
				return nil, anError
			},
		},
		{
			name: "nil system cert pool",
			tlsConfig: Config{
				IncludeSystemCACertsPool: true,
				CAFile:                   filepath.Join("testdata", "ca-1.crt"),
			},
			wantErr: nil,
			systemCertFn: func() (*x509.CertPool, error) {
				return nil, nil
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			oldSystemCertPool := systemCertPool
			systemCertPool = test.systemCertFn
			defer func() {
				systemCertPool = oldSystemCertPool
			}()

			serverConfig := ServerConfig{
				Config: test.tlsConfig,
			}
			c, err := serverConfig.LoadTLSConfig(context.Background())
			if test.wantErr != nil {
				require.ErrorContains(t, err, test.wantErr.Error())
			} else {
				assert.NotNil(t, c.RootCAs)
			}

			clientConfig := ClientConfig{
				Config: test.tlsConfig,
			}
			c, err = clientConfig.LoadTLSConfig(context.Background())
			if test.wantErr != nil {
				assert.ErrorContains(t, err, test.wantErr.Error())
			} else {
				assert.NotNil(t, c.RootCAs)
			}
		})
	}
}

func TestSystemCertPool_loadCert(t *testing.T) {
	anError := errors.New("my error")
	tests := []struct {
		name         string
		tlsConfig    Config
		wantErr      error
		systemCertFn func() (*x509.CertPool, error)
	}{
		{
			name: "not using system cert pool",
			tlsConfig: Config{
				IncludeSystemCACertsPool: false,
			},
			wantErr:      nil,
			systemCertFn: x509.SystemCertPool,
		},
		{
			name: "using system cert pool",
			tlsConfig: Config{
				IncludeSystemCACertsPool: true,
			},
			wantErr:      nil,
			systemCertFn: x509.SystemCertPool,
		},
		{
			name: "error loading system cert pool",
			tlsConfig: Config{
				IncludeSystemCACertsPool: true,
			},
			wantErr: anError,
			systemCertFn: func() (*x509.CertPool, error) {
				return nil, anError
			},
		},
		{
			name: "nil system cert pool",
			tlsConfig: Config{
				IncludeSystemCACertsPool: true,
			},
			wantErr: nil,
			systemCertFn: func() (*x509.CertPool, error) {
				return nil, nil
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			oldSystemCertPool := systemCertPool
			systemCertPool = test.systemCertFn
			defer func() {
				systemCertPool = oldSystemCertPool
			}()
			certPool, err := test.tlsConfig.loadCert(filepath.Join("testdata", "ca-1.crt"))
			if test.wantErr != nil {
				assert.Equal(t, test.wantErr, err)
			} else {
				assert.NotNil(t, certPool)
			}
		})
	}
}

func TestCurvePreferences(t *testing.T) {
	type testCase struct {
		name             string
		preferences      []string
		expectedCurveIDs []tls.CurveID
		expectedErr      string
	}

	tests := []testCase{
		{
			name:             "P521",
			preferences:      []string{"P521"},
			expectedCurveIDs: []tls.CurveID{tls.CurveP521},
		},
		{
			name:             "P-256",
			preferences:      []string{"P256"},
			expectedCurveIDs: []tls.CurveID{tls.CurveP256},
		},
		{
			name:             "multiple",
			preferences:      []string{"P256", "P521"},
			expectedCurveIDs: []tls.CurveID{tls.CurveP256, tls.CurveP521},
		},
		{
			name:             "invalid-curve",
			preferences:      []string{"P25223236"},
			expectedCurveIDs: []tls.CurveID{},
			expectedErr:      "invalid curve type",
		},
	}

	// X25519 curves are not supported when GODEBUG=fips140=only is set, so we
	// detect if it is and conditionally add test cases for those curves.
	if !strings.Contains(os.Getenv("GODEBUG"), "fips140=only") {
		tests = append(tests,
			testCase{
				name:             "X25519MLKEM768",
				preferences:      []string{"X25519MLKEM768"},
				expectedCurveIDs: []tls.CurveID{tls.X25519MLKEM768},
			},
			testCase{
				name:             "X25519",
				preferences:      []string{"X25519"},
				expectedCurveIDs: []tls.CurveID{tls.X25519},
			},
		)
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tlsSetting := ClientConfig{
				Config: Config{
					CurvePreferences: test.preferences,
				},
			}
			config, err := tlsSetting.LoadTLSConfig(context.Background())
			if test.expectedErr == "" {
				require.NoError(t, err)
				require.ElementsMatchf(t, test.expectedCurveIDs, config.CurvePreferences, "expected %v, got %v", test.expectedCurveIDs, config.CurvePreferences)
			} else {
				require.ErrorContains(t, err, test.expectedErr)
			}
		})
	}
}

func TestServerConfigValidate(t *testing.T) {
	tests := []struct {
		name         string
		serverConfig ServerConfig
		errorTxt     string
	}{
		{
			name:         "server config without certificates",
			serverConfig: ServerConfig{},
			errorTxt:     "TLS configuration must include both certificate and key for server connections",
		},
		{
			name: "server config with cert file but no key",
			serverConfig: ServerConfig{
				Config: Config{
					CertFile: "cert.pem",
				},
			},
			errorTxt: "config: TLS configuration must include both certificate and key (CertFile/CertPem and KeyFile/KeyPem)",
		},
		{
			name: "server config with key file but no cert",
			serverConfig: ServerConfig{
				Config: Config{
					KeyFile: "key.pem",
				},
			},
			errorTxt: "config: TLS configuration must include both certificate and key (CertFile/CertPem and KeyFile/KeyPem)",
		},
		{
			name: "server config with cert PEM but no key",
			serverConfig: ServerConfig{
				Config: Config{
					CertPem: "cert-pem",
				},
			},
			errorTxt: "config: TLS configuration must include both certificate and key (CertFile/CertPem and KeyFile/KeyPem)",
		},
		{
			name: "server config with key PEM but no cert",
			serverConfig: ServerConfig{
				Config: Config{
					KeyPem: "key-pem",
				},
			},
			errorTxt: "config: TLS configuration must include both certificate and key (CertFile/CertPem and KeyFile/KeyPem)",
		},
		{
			name: "server config with both cert file and cert PEM",
			serverConfig: ServerConfig{
				Config: Config{
					CertFile: "cert.pem",
					CertPem:  "cert-pem",
					KeyFile:  "key.pem",
				},
			},
			errorTxt: "config: provide either certificate file or PEM, but not both",
		},
		{
			name: "server config with both key file and key PEM",
			serverConfig: ServerConfig{
				Config: Config{
					CertFile: "cert.pem",
					KeyFile:  "key.pem",
					KeyPem:   "key-pem",
				},
			},
			errorTxt: "config: provide either key file or PEM, but not both",
		},
		{
			name: "valid server config with cert and key files",
			serverConfig: ServerConfig{
				Config: Config{
					CertFile: "cert.pem",
					KeyFile:  "key.pem",
				},
			},
		},
		{
			name: "valid server config with cert and key PEM",
			serverConfig: ServerConfig{
				Config: Config{
					CertPem: "cert-pem",
					KeyPem:  "key-pem",
				},
			},
		},
		{
			name: "valid server config with mixed cert file and key PEM",
			serverConfig: ServerConfig{
				Config: Config{
					CertFile: "cert.pem",
					KeyPem:   "key-pem",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := xconfmap.Validate(test.serverConfig)

			if test.errorTxt == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, test.errorTxt)
			}
		})
	}
}

func TestClientConfigValidate(t *testing.T) {
	tests := []struct {
		name         string
		clientConfig ClientConfig
		errorTxt     string
	}{
		{
			name:         "valid empty client config",
			clientConfig: ClientConfig{},
		},
		{
			name: "valid client config with insecure connection",
			clientConfig: ClientConfig{
				Insecure: true,
			},
		},
		{
			name: "valid client config with cert and key files",
			clientConfig: ClientConfig{
				Config: Config{
					CertFile: "cert.pem",
					KeyFile:  "key.pem",
				},
			},
		},
		{
			name: "valid client config with mixed cert file and key PEM",
			clientConfig: ClientConfig{
				Config: Config{
					CertFile: "cert.pem",
					KeyPem:   "key-pem",
				},
			},
		},
		{
			name: "client config with only cert file",
			clientConfig: ClientConfig{
				Config: Config{
					CertFile: "cert.pem",
				},
			},
			errorTxt: "config: TLS configuration must include both certificate and key (CertFile/CertPem and KeyFile/KeyPem)",
		},
		{
			name: "client config with only key file",
			clientConfig: ClientConfig{
				Config: Config{
					KeyFile: "key.pem",
				},
			},
			errorTxt: "config: TLS configuration must include both certificate and key (CertFile/CertPem and KeyFile/KeyPem)",
		},
		{
			name: "client config with only cert PEM",
			clientConfig: ClientConfig{
				Config: Config{
					CertPem: "cert-pem",
				},
			},
			errorTxt: "config: TLS configuration must include both certificate and key (CertFile/CertPem and KeyFile/KeyPem)",
		},
		{
			name: "client config with only key PEM",
			clientConfig: ClientConfig{
				Config: Config{
					KeyPem: "key-pem",
				},
			},
			errorTxt: "config: TLS configuration must include both certificate and key (CertFile/CertPem and KeyFile/KeyPem)",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := xconfmap.Validate(test.clientConfig)

			if test.errorTxt == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, test.errorTxt)
			}
		})
	}
}
