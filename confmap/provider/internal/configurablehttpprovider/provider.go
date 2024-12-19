// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configurablehttpprovider // import "go.opentelemetry.io/collector/confmap/provider/internal/configurablehttpprovider"

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"go.opentelemetry.io/collector/confmap"
)

type SchemeType string

const (
	HTTPScheme  SchemeType = "http"
	HTTPSScheme SchemeType = "https"
)

type provider struct {
	scheme             SchemeType
	caCertPath         string // Used for tests
	insecureSkipVerify bool   // Used for tests
}

// New returns a new provider that reads the configuration from http server using the configured transport mechanism
// depending on the selected scheme.
// There are two types of transport supported: PlainText (HTTPScheme) and TLS (HTTPSScheme).
//
// One example for http-uri: http://localhost:3333/getConfig
// One example for https-uri: https://localhost:3333/getConfig
// This is used by the http and https external implementations.
func New(scheme SchemeType, _ confmap.ProviderSettings) confmap.Provider {
	return &provider{scheme: scheme}
}

// Create the client based on the type of scheme that was selected.
func (fmp *provider) createClient() (*http.Client, error) {
	switch fmp.scheme {
	case HTTPScheme:
		return &http.Client{}, nil
	case HTTPSScheme:
		pool, err := x509.SystemCertPool()
		if err != nil {
			return nil, fmt.Errorf("unable to create a cert pool: %w", err)
		}

		if fmp.caCertPath != "" {
			cert, err := os.ReadFile(filepath.Clean(fmp.caCertPath))
			if err != nil {
				return nil, fmt.Errorf("unable to read CA from %q URI: %w", fmp.caCertPath, err)
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
	default:
		return nil, fmt.Errorf("invalid scheme type: %s", fmp.scheme)
	}
}

func (fmp *provider) Retrieve(_ context.Context, uri string, _ confmap.WatcherFunc) (*confmap.Retrieved, error) {
	if !strings.HasPrefix(uri, string(fmp.scheme)+":") {
		return nil, fmt.Errorf("%q uri is not supported by %q provider", uri, string(fmp.scheme))
	}

	if _, err := url.ParseRequestURI(uri); err != nil {
		return nil, fmt.Errorf("invalid uri %q: %w", uri, err)
	}

	client, err := fmp.createClient()
	if err != nil {
		return nil, fmt.Errorf("unable to configure http transport layer: %w", err)
	}

	// send a HTTP GET request
	resp, err := client.Get(uri)
	if err != nil {
		return nil, fmt.Errorf("unable to download the file via HTTP GET for uri %q: %w ", uri, err)
	}
	defer resp.Body.Close()

	// check the HTTP status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to load resource from uri %q. status code: %d", uri, resp.StatusCode)
	}

	// read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("fail to read the response body from uri %q: %w", uri, err)
	}

	return confmap.NewRetrievedFromYAML(body)
}

func (fmp *provider) Scheme() string {
	return string(fmp.scheme)
}

func (*provider) Shutdown(context.Context) error {
	return nil
}
