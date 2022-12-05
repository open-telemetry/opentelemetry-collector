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

package configurablehttpprovider // import "go.opentelemetry.io/collector/confmap/provider/internal/configurablehttpprovider"

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/internal"
)

type TransportType string

const (
	PlainText TransportType = "http"
	TLS       TransportType = "https"
)

type provider struct {
	transport          TransportType
	caCertPath         string // Used for tests
	insecureSkipVerify bool   // Used for tests
}

// New returns a new provider that reads the configuration from http server using the configured transport mechanism.
// There are two types of transport supported: PlainText (HTTP) and TLS (HTTPS).
//
// This internal provider supports "http" and "https" schemes based on the transport chosen, and can be called with a "uri" that
// follows:
//
// One example for http-uri: http://localhost:3333/getConfig
// One example for https-uri: https://localhost:3333/getConfig
// This is used by the http and https external implementations.
func New(transport TransportType) confmap.Provider {
	return newConfigurableHTTPProvider(transport)
}

func newConfigurableHTTPProvider(transport TransportType) *provider {
	return &provider{transport: transport, caCertPath: "", insecureSkipVerify: false}
}

func (fmp *provider) getHTTPClient() (*http.Client, error) {
	return &http.Client{}, nil

}

func (fmp *provider) getHTTPSClient() (*http.Client, error) {
	pool, err := x509.SystemCertPool()

	if err != nil {
		return nil, fmt.Errorf("unable to create a cert pool: %w", err)
	}

	if fmp.caCertPath != "" {
		cert, err := os.ReadFile(filepath.Clean(fmp.caCertPath))

		if err != nil {
			return nil, fmt.Errorf("unable to read CA from uri: %s", fmp.caCertPath)
		}

		if ok := pool.AppendCertsFromPEM(cert); !ok {
			return nil, fmt.Errorf("unable to add CA from uri: %s into the cert pool", fmp.caCertPath)
		}
	}

	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: fmp.insecureSkipVerify,
				RootCAs:            pool,
			},
		},
	}, nil
}

// Get the client based on the type of transport that was selected.
func (fmp *provider) getClient() (*http.Client, error) {
	switch fmp.transport {
	case PlainText:
		return fmp.getHTTPClient()
	case TLS:
		return fmp.getHTTPSClient()
	default:
		return nil, fmt.Errorf("invalid transport type: %s", fmp.transport)
	}
}

func (fmp *provider) Retrieve(_ context.Context, uri string, _ confmap.WatcherFunc) (*confmap.Retrieved, error) {

	if !strings.HasPrefix(uri, string(fmp.transport)+":") {
		return nil, fmt.Errorf("%q uri is not supported by %q provider", uri, string(fmp.transport))
	}

	client, err := fmp.getClient()

	if err != nil {
		return nil, fmt.Errorf("unable to configure http transport layer: %w", err)
	}

	// send a HTTP GET request
	resp, err := client.Get(uri)
	if err != nil {
		return nil, fmt.Errorf("unable to download the file via HTTP GET for uri %q, with err: %w ", uri, err)
	}
	defer resp.Body.Close()

	// check the HTTP status code
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to load resource from uri %q. status code: %d", uri, resp.StatusCode)
	}

	// read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("fail to read the response body from uri %q, with err: %w ", uri, err)
	}

	return internal.NewRetrievedFromYAML(body)
}

func (fmp *provider) Scheme() string {
	return string(fmp.transport)
}

func (*provider) Shutdown(context.Context) error {
	return nil
}
