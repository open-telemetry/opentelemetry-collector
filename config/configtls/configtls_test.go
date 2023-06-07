// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configtls

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config/configopaque"
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
			options: TLSSetting{CAFile: filepath.Join("testdata", "ca-1.crt")},
		},
		{
			name:        "should fail with invalid CA file path",
			options:     TLSSetting{CAFile: filepath.Join("testdata", "not/valid")},
			expectError: "failed to load CA",
		},
		{
			name:        "should fail with invalid CA file content",
			options:     TLSSetting{CAFile: filepath.Join("testdata", "testCA-bad.txt")},
			expectError: "failed to parse cert",
		},
		{
			name: "should load valid TLS  settings",
			options: TLSSetting{
				CAFile:   filepath.Join("testdata", "ca-1.crt"),
				CertFile: filepath.Join("testdata", "server-1.crt"),
				KeyFile:  filepath.Join("testdata", "server-1.key"),
			},
		},
		{
			name: "should fail with missing TLS KeyFile",
			options: TLSSetting{
				CAFile:   filepath.Join("testdata", "ca-1.crt"),
				CertFile: filepath.Join("testdata", "server-1.crt"),
			},
			expectError: "provide both certificate and key, or neither",
		},
		{
			name: "should fail with invalid TLS KeyFile",
			options: TLSSetting{
				CAFile:   filepath.Join("testdata", "ca-1.crt"),
				CertFile: filepath.Join("testdata", "server-1.crt"),
				KeyFile:  filepath.Join("testdata", "not/valid"),
			},
			expectError: "failed to load TLS cert and key",
		},
		{
			name: "should fail with missing TLS Cert",
			options: TLSSetting{
				CAFile:  filepath.Join("testdata", "ca-1.crt"),
				KeyFile: filepath.Join("testdata", "server-1.key"),
			},
			expectError: "provide both certificate and key, or neither",
		},
		{
			name: "should fail with invalid TLS Cert",
			options: TLSSetting{
				CAFile:   filepath.Join("testdata", "ca-1.crt"),
				CertFile: filepath.Join("testdata", "not/valid"),
				KeyFile:  filepath.Join("testdata", "server-1.key"),
			},
			expectError: "failed to load TLS cert and key",
		},
		{
			name: "should fail with invalid TLS CA",
			options: TLSSetting{
				CAFile: filepath.Join("testdata", "not/valid"),
			},
			expectError: "failed to load CA",
		},
		{
			name: "should fail with invalid CA pool",
			options: TLSSetting{
				CAFile: filepath.Join("testdata", "testCA-bad.txt"),
			},
			expectError: "failed to parse cert",
		},
		{
			name: "should pass with valid CA pool",
			options: TLSSetting{
				CAFile: filepath.Join("testdata", "ca-1.crt"),
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
		{
			name:    "should load custom CA PEM",
			options: TLSSetting{CAPem: readFilePanics("testdata/ca-1.crt")},
		},
		{
			name: "should load valid TLS settings with PEMs",
			options: TLSSetting{
				CAPem:   readFilePanics("testdata/ca-1.crt"),
				CertPem: readFilePanics("testdata/server-1.crt"),
				KeyPem:  readFilePanics("testdata/server-1.key"),
			},
		},
		{
			name: "mix Cert file and Key PEM provided",
			options: TLSSetting{
				CertFile: "testdata/server-1.crt",
				KeyPem:   readFilePanics("testdata/server-1.key"),
			},
		},
		{
			name: "mix Cert PEM and Key File provided",
			options: TLSSetting{
				CertPem: readFilePanics("testdata/server-1.crt"),
				KeyFile: "testdata/server-1.key",
			},
		},
		{
			name:        "should fail with invalid CA PEM",
			options:     TLSSetting{CAPem: readFilePanics("testdata/testCA-bad.txt")},
			expectError: "failed to parse cert",
		},
		{
			name: "should fail CA file and PEM both provided",
			options: TLSSetting{
				CAFile: "testdata/ca-1.crt",
				CAPem:  readFilePanics("testdata/ca-1.crt"),
			},
			expectError: "provide either a CA file or the PEM-encoded string, but not both",
		},
		{
			name: "should fail Cert file and PEM both provided",
			options: TLSSetting{
				CertFile: "testdata/server-1.crt",
				CertPem:  readFilePanics("testdata/server-1.crt"),
				KeyFile:  "testdata/server-1.key",
			},
			expectError: "provide either a certificate or the PEM-encoded string, but not both",
		},
		{
			name: "should fail Key file and PEM both provided",
			options: TLSSetting{
				CertFile: "testdata/server-1.crt",
				KeyFile:  "testdata/ca-1.crt",
				KeyPem:   readFilePanics("testdata/server-1.key"),
			},
			expectError: "provide either a key or the PEM-encoded string, but not both",
		},
		{
			name: "should fail to load valid TLS settings with bad Cert PEM",
			options: TLSSetting{
				CAPem:   readFilePanics("testdata/ca-1.crt"),
				CertPem: readFilePanics("testdata/testCA-bad.txt"),
				KeyPem:  readFilePanics("testdata/server-1.key"),
			},
			expectError: "failed to load TLS cert and key PEMs",
		},
		{
			name: "should fail to load valid TLS settings with bad Key PEM",
			options: TLSSetting{
				CAPem:   readFilePanics("testdata/ca-1.crt"),
				CertPem: readFilePanics("testdata/server-1.crt"),
				KeyPem:  readFilePanics("testdata/testCA-bad.txt"),
			},
			expectError: "failed to load TLS cert and key PEMs",
		},
		{
			name: "should fail with missing TLS KeyPem",
			options: TLSSetting{
				CAPem:   readFilePanics("testdata/ca-1.crt"),
				CertPem: readFilePanics("testdata/server-1.crt"),
			},
			expectError: "provide both certificate and key, or neither",
		},
		{
			name: "should fail with missing TLS Cert PEM",
			options: TLSSetting{
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
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.expectError)
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

func TestLoadTLSServerConfigReload(t *testing.T) {

	tmpCaPath := createTempClientCaFile(t)

	overwriteClientCA(t, tmpCaPath, "ca-1.crt")

	tlsSetting := TLSServerSetting{
		ClientCAFile:       tmpCaPath,
		ReloadClientCAFile: true,
	}

	tlsCfg, err := tlsSetting.LoadTLSConfig()
	assert.NoError(t, err)
	assert.NotNil(t, tlsCfg)

	firstClient, err := tlsCfg.GetConfigForClient(nil)
	assert.NoError(t, err)

	overwriteClientCA(t, tmpCaPath, "ca-2.crt")

	assert.Eventually(t, func() bool {
		_, loadError := tlsCfg.GetConfigForClient(nil)
		return loadError == nil
	}, 5*time.Second, 10*time.Millisecond)

	secondClient, err := tlsCfg.GetConfigForClient(nil)
	assert.NoError(t, err)

	assert.NotEqual(t, firstClient.ClientCAs, secondClient.ClientCAs)
}

func TestLoadTLSServerConfigFailingReload(t *testing.T) {

	tmpCaPath := createTempClientCaFile(t)

	overwriteClientCA(t, tmpCaPath, "ca-1.crt")

	tlsSetting := TLSServerSetting{
		ClientCAFile:       tmpCaPath,
		ReloadClientCAFile: true,
	}

	tlsCfg, err := tlsSetting.LoadTLSConfig()
	assert.NoError(t, err)
	assert.NotNil(t, tlsCfg)

	firstClient, err := tlsCfg.GetConfigForClient(nil)
	assert.NoError(t, err)

	overwriteClientCA(t, tmpCaPath, "testCA-bad.txt")

	assert.Eventually(t, func() bool {
		_, loadError := tlsCfg.GetConfigForClient(nil)
		return loadError == nil
	}, 5*time.Second, 10*time.Millisecond)

	secondClient, err := tlsCfg.GetConfigForClient(nil)
	assert.NoError(t, err)

	assert.Equal(t, firstClient.ClientCAs, secondClient.ClientCAs)
}

func TestLoadTLSServerConfigFailingInitialLoad(t *testing.T) {

	tmpCaPath := createTempClientCaFile(t)

	overwriteClientCA(t, tmpCaPath, "testCA-bad.txt")

	tlsSetting := TLSServerSetting{
		ClientCAFile:       tmpCaPath,
		ReloadClientCAFile: true,
	}

	tlsCfg, err := tlsSetting.LoadTLSConfig()
	assert.Error(t, err)
	assert.Nil(t, tlsCfg)
}

func TestLoadTLSServerConfigWrongPath(t *testing.T) {

	tmpCaPath := createTempClientCaFile(t)

	tlsSetting := TLSServerSetting{
		ClientCAFile:       tmpCaPath + "wrong-path",
		ReloadClientCAFile: true,
	}

	tlsCfg, err := tlsSetting.LoadTLSConfig()
	assert.Error(t, err)
	assert.Nil(t, tlsCfg)
}

func TestLoadTLSServerConfigFailing(t *testing.T) {

	tmpCaPath := createTempClientCaFile(t)

	overwriteClientCA(t, tmpCaPath, "ca-1.crt")

	tlsSetting := TLSServerSetting{
		ClientCAFile:       tmpCaPath,
		ReloadClientCAFile: true,
	}

	tlsCfg, err := tlsSetting.LoadTLSConfig()
	assert.NoError(t, err)
	assert.NotNil(t, tlsCfg)

	firstClient, err := tlsCfg.GetConfigForClient(nil)
	assert.NoError(t, err)
	assert.NotNil(t, firstClient)

	err = os.Remove(tmpCaPath)
	assert.NoError(t, err)

	firstClient, err = tlsCfg.GetConfigForClient(nil)
	assert.NoError(t, err)
	assert.NotNil(t, firstClient)
}

func overwriteClientCA(t *testing.T, targetFilePath string, testdataFileName string) {
	targetFile, err := os.OpenFile(filepath.Clean(targetFilePath), os.O_RDWR, 0600)
	assert.NoError(t, err)

	testdataFilePath := filepath.Join("testdata", testdataFileName)
	testdataFile, err := os.OpenFile(filepath.Clean(testdataFilePath), os.O_RDONLY, 0200)
	assert.NoError(t, err)

	_, err = io.Copy(targetFile, testdataFile)
	assert.NoError(t, err)

	assert.NoError(t, targetFile.Close())
	assert.NoError(t, testdataFile.Close())
}

func createTempClientCaFile(t *testing.T) string {
	tmpCa, err := os.CreateTemp("", "ca-tmp.crt")
	assert.NoError(t, err)
	tmpCaPath, err := filepath.Abs(tmpCa.Name())
	assert.NoError(t, err)
	assert.NoError(t, tmpCa.Close())
	return tmpCaPath
}

func TestEagerlyLoadCertificate(t *testing.T) {
	options := TLSSetting{
		CertFile: filepath.Join("testdata", "client-1.crt"),
		KeyFile:  filepath.Join("testdata", "client-1.key"),
	}
	cfg, err := options.loadTLSConfig()
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	cert, err := cfg.GetCertificate(&tls.ClientHelloInfo{})
	assert.NoError(t, err)
	assert.NotNil(t, cert)
	pCert, err := x509.ParseCertificate(cert.Certificate[0])
	assert.NoError(t, err)
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
			certFile, err := os.CreateTemp("", "cert")
			assert.NoError(t, err)
			defer os.Remove(certFile.Name())

			keyFile, err := os.CreateTemp("", "key")
			assert.NoError(t, err)
			defer os.Remove(keyFile.Name())

			fdc, err := os.Open(filepath.Join("testdata", "client-1.crt"))
			assert.NoError(t, err)
			_, err = io.Copy(certFile, fdc)
			assert.NoError(t, err)

			fdk, err := os.Open(filepath.Join("testdata", "client-1.key"))
			assert.NoError(t, err)
			_, err = io.Copy(keyFile, fdk)
			assert.NoError(t, err)
			assert.NoError(t, fdk.Close())

			options := TLSSetting{
				CertFile:       certFile.Name(),
				KeyFile:        keyFile.Name(),
				ReloadInterval: test.reloadInterval,
			}
			cfg, err := options.loadTLSConfig()
			assert.NoError(t, err)
			assert.NotNil(t, cfg)

			// Asssert that we loaded the original certificate
			cert, err := cfg.GetCertificate(&tls.ClientHelloInfo{})
			assert.NoError(t, err)
			assert.NotNil(t, cert)
			pCert, err := x509.ParseCertificate(cert.Certificate[0])
			assert.NoError(t, err)
			assert.NotNil(t, pCert)
			assert.Equal(t, test.dns1, pCert.DNSNames[0])

			// Change the certificate
			assert.NoError(t, certFile.Truncate(0))
			assert.NoError(t, keyFile.Truncate(0))
			_, err = certFile.Seek(0, 0)
			assert.NoError(t, err)
			_, err = keyFile.Seek(0, 0)
			assert.NoError(t, err)

			fdc2, err := os.Open(filepath.Join("testdata", test.cert2))
			assert.NoError(t, err)
			_, err = io.Copy(certFile, fdc2)
			assert.NoError(t, err)
			assert.NoError(t, fdc2.Close())

			fdk2, err := os.Open(filepath.Join("testdata", test.key2))
			assert.NoError(t, err)
			_, err = io.Copy(keyFile, fdk2)
			assert.NoError(t, err)
			assert.NoError(t, fdk2.Close())

			// Wait ReloadInterval to ensure a reload will happen
			time.Sleep(test.wait)

			// Assert that we loaded the new certificate
			cert, err = cfg.GetCertificate(&tls.ClientHelloInfo{})
			if test.errText == "" {
				assert.NoError(t, err)
				assert.NotNil(t, cert)
				pCert, err = x509.ParseCertificate(cert.Certificate[0])
				assert.NoError(t, err)
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
			setting := TLSSetting{
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
