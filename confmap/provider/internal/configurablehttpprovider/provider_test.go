// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package configurablehttpprovider

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/confmap/confmaptest"
)

func newConfigurableHTTPProvider(scheme SchemeType) *provider {
	return New(scheme).(*provider)
}

func answerGet(w http.ResponseWriter, _ *http.Request) {
	f, err := os.ReadFile("./testdata/otel-config.yaml")
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_, innerErr := w.Write([]byte("Cannot find the config file"))
		if innerErr != nil {
			fmt.Println("Write failed: ", innerErr)
		}
		return
	}
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(f)
	if err != nil {
		fmt.Println("Write failed: ", err)
	}
}

// Generate a self signed certificate specific for the tests. Based on
// https://go.dev/src/crypto/tls/generate_cert.go
func generateCertificate(hostname string) (cert string, key string, err error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)

	if err != nil {
		return "", "", fmt.Errorf("Failed to generate private key: %w", err)
	}

	keyUsage := x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign
	notBefore := time.Now()
	notAfter := notBefore.Add(time.Hour * 12)
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)

	if err != nil {
		return "", "", fmt.Errorf("Failed to generate serial number: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Httpprovider Co"},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              keyUsage,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		DNSNames:              []string{hostname},
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)

	if err != nil {
		return "", "", fmt.Errorf("Failed to create certificate: %w", err)
	}

	certOut, err := os.CreateTemp("", "cert*.pem")
	if err != nil {
		return "", "", fmt.Errorf("Failed to open cert.pem for writing: %w", err)
	}

	if err = pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return "", "", fmt.Errorf("Failed to write data to cert.pem: %w", err)
	}

	if err = certOut.Close(); err != nil {
		return "", "", fmt.Errorf("Error closing cert.pem: %w", err)
	}

	keyOut, err := os.CreateTemp("", "key*.pem")

	if err != nil {
		return "", "", fmt.Errorf("Failed to open key.pem for writing: %w", err)
	}

	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)

	if err != nil {
		return "", "", fmt.Errorf("Unable to marshal private key: %w", err)
	}

	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes}); err != nil {
		return "", "", fmt.Errorf("Failed to write data to key.pem: %w", err)
	}

	if err := keyOut.Close(); err != nil {
		return "", "", fmt.Errorf("Error closing key.pem: %w", err)
	}

	return certOut.Name(), keyOut.Name(), nil
}

func TestFunctionalityDownloadFileHTTP(t *testing.T) {
	fp := newConfigurableHTTPProvider(HTTPScheme)
	ts := httptest.NewServer(http.HandlerFunc(answerGet))
	defer ts.Close()
	_, err := fp.Retrieve(context.Background(), ts.URL, nil)
	assert.NoError(t, err)
	assert.NoError(t, fp.Shutdown(context.Background()))
}

func TestFunctionalityDownloadFileHTTPS(t *testing.T) {
	certPath, keyPath, err := generateCertificate("localhost")
	assert.NoError(t, err)

	invalidCert, err := os.CreateTemp("", "cert*.crt")
	assert.NoError(t, err)
	_, err = invalidCert.Write([]byte{0, 1, 2})
	assert.NoError(t, err)

	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	assert.NoError(t, err)
	ts := httptest.NewUnstartedServer(http.HandlerFunc(answerGet))
	ts.TLS = &tls.Config{Certificates: []tls.Certificate{cert}}
	ts.StartTLS()

	defer os.Remove(certPath)
	defer os.Remove(keyPath)
	defer os.Remove(invalidCert.Name())
	defer ts.Close()

	tests := []struct {
		testName               string
		certPath               string
		hostName               string
		useCertificate         bool
		skipHostnameValidation bool
		shouldError            bool
	}{
		{
			testName:               "Test valid certificate and name",
			certPath:               certPath,
			hostName:               "localhost",
			useCertificate:         true,
			skipHostnameValidation: false,
			shouldError:            false,
		},
		{
			testName:               "Test valid certificate with invalid name",
			certPath:               certPath,
			hostName:               "127.0.0.1",
			useCertificate:         true,
			skipHostnameValidation: false,
			shouldError:            true,
		},
		{
			testName:               "Test valid certificate with invalid name, skip validation",
			certPath:               certPath,
			hostName:               "127.0.0.1",
			useCertificate:         true,
			skipHostnameValidation: true,
			shouldError:            false,
		},
		{
			testName:               "Test no certificate should fail",
			certPath:               certPath,
			hostName:               "localhost",
			useCertificate:         false,
			skipHostnameValidation: false,
			shouldError:            true,
		},
		{
			testName:               "Test invalid cert",
			certPath:               invalidCert.Name(),
			hostName:               "localhost",
			useCertificate:         true,
			skipHostnameValidation: false,
			shouldError:            true,
		},
		{
			testName:               "Test no cert",
			certPath:               "no_certificate",
			hostName:               "localhost",
			useCertificate:         true,
			skipHostnameValidation: false,
			shouldError:            true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			fp := newConfigurableHTTPProvider(HTTPSScheme)
			// Parse url of the test server to get the port number.
			tsURL, err := url.Parse(ts.URL)
			require.NoError(t, err)
			if tt.useCertificate {
				fp.caCertPath = tt.certPath
			}
			fp.insecureSkipVerify = tt.skipHostnameValidation
			_, err = fp.Retrieve(context.Background(), fmt.Sprintf("https://%s:%s", tt.hostName, tsURL.Port()), nil)
			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUnsupportedScheme(t *testing.T) {
	fp := New(HTTPScheme)
	_, err := fp.Retrieve(context.Background(), "https://...", nil)
	assert.Error(t, err)
	assert.NoError(t, fp.Shutdown(context.Background()))

	fp = New(HTTPSScheme)
	_, err = fp.Retrieve(context.Background(), "http://...", nil)
	assert.Error(t, err)
	assert.NoError(t, fp.Shutdown(context.Background()))
}

func TestEmptyURI(t *testing.T) {
	fp := New(HTTPScheme)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer ts.Close()
	_, err := fp.Retrieve(context.Background(), ts.URL, nil)
	require.Error(t, err)
	require.NoError(t, fp.Shutdown(context.Background()))
}

func TestRetrieveFromShutdownServer(t *testing.T) {
	fp := New(HTTPScheme)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	ts.Close()
	_, err := fp.Retrieve(context.Background(), ts.URL, nil)
	assert.Error(t, err)
	require.NoError(t, fp.Shutdown(context.Background()))
}

func TestNonExistent(t *testing.T) {
	fp := New(HTTPScheme)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()
	_, err := fp.Retrieve(context.Background(), ts.URL, nil)
	assert.Error(t, err)
	require.NoError(t, fp.Shutdown(context.Background()))
}

func TestInvalidYAML(t *testing.T) {
	fp := New(HTTPScheme)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("wrong : ["))
		if err != nil {
			fmt.Println("Write failed: ", err)
		}
	}))
	defer ts.Close()
	_, err := fp.Retrieve(context.Background(), ts.URL, nil)
	assert.Error(t, err)
	require.NoError(t, fp.Shutdown(context.Background()))
}

func TestScheme(t *testing.T) {
	fp := New(HTTPScheme)
	assert.Equal(t, "http", fp.Scheme())
	require.NoError(t, fp.Shutdown(context.Background()))
}

func TestValidateProviderScheme(t *testing.T) {
	assert.NoError(t, confmaptest.ValidateProviderScheme(New(HTTPScheme)))
}

func TestInvalidTransport(t *testing.T) {
	fp := New("foo")

	_, err := fp.Retrieve(context.Background(), "foo://..", nil)
	assert.Error(t, err)
}
