// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configurablehttpprovider // import "go.opentelemetry.io/collector/confmap/provider/internal/configurablehttpprovider"

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
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

func (fmp *provider) Retrieve(ctx context.Context, uri string, _ confmap.WatcherFunc) (*confmap.Retrieved, error) {
	if !strings.HasPrefix(uri, string(fmp.scheme)+":") {
		return nil, fmt.Errorf("%q uri is not supported by %q provider", uri, string(fmp.scheme))
	}

	parsedURI, err := url.ParseRequestURI(uri)
	if err != nil {
		// The uri may carry credentials (http://user:password@host/path) and cannot be
		// redacted when it does not parse. *url.Error quotes the raw uri, so report only
		// the underlying reason.
		if urlErr, ok := errors.AsType[*url.Error](err); ok {
			err = urlErr.Err
		}
		return nil, fmt.Errorf("invalid uri: %w", err)
	}
	// Userinfo in the uri is sent as basic auth; never put the password in an error,
	// which ends up on stderr or in logs.
	redactedURI := parsedURI.Redacted()

	client, err := fmp.createClient()
	if err != nil {
		return nil, fmt.Errorf("unable to configure http transport layer: %w", err)
	}

	// send a HTTP GET request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("unable to create HTTP GET request for uri %q: %w", redactedURI, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unable to download the file via HTTP GET for uri %q: %w", redactedURI, err)
	}
	defer resp.Body.Close()

	// check the HTTP status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to load resource from uri %q. status code: %d", redactedURI, resp.StatusCode)
	}

	// read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("fail to read the response body from uri %q: %w", redactedURI, err)
	}

	return confmap.NewRetrievedFromYAML(body)
}

func (fmp *provider) Scheme() string {
	return string(fmp.scheme)
}

func (*provider) Shutdown(context.Context) error {
	return nil
}
