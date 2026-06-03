// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configurablehttpprovider // import "go.opentelemetry.io/collector/confmap/provider/internal/configurablehttpprovider"

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/confmap"
)

type SchemeType string

const (
	HTTPScheme  SchemeType = "http"
	HTTPSScheme SchemeType = "https"
)

// pollingIntervalQueryParam is the name of the URI query parameter
// that opts the provider into polling the configured URL for
// changes. The name is namespaced with an "otel_config_" prefix to
// make collisions with a query parameter the upstream server already
// uses very unlikely.
const pollingIntervalQueryParam = "otel_config_polling_interval"

type provider struct {
	scheme             SchemeType
	caCertPath         string // Used for tests
	insecureSkipVerify bool   // Used for tests
	logger             *zap.Logger

	mu sync.Mutex

	// nextID hands out a unique key for each in-flight polling Retrieve so its
	// Retrieved.Close can remove exactly its own cancel from the map below.
	nextID uint64

	// cancels holds the cancel func for every currently-active polling
	// goroutine, keyed by nextID. Entries are added in Retrieve and removed in
	// the matching Retrieved.Close, so the map stays bounded to the number of
	// live config sources even across the many Retrieve/Close cycles a
	// long-running Collector performs on every config reload.
	cancels map[uint64]context.CancelFunc
	wg      sync.WaitGroup
}

// New returns a new provider that reads the configuration from http server using the configured transport mechanism
// depending on the selected scheme.
// There are two types of transport supported: PlainText (HTTPScheme) and TLS (HTTPSScheme).
//
// One example for http-uri: http://localhost:3333/getConfig
// One example for https-uri: https://localhost:3333/getConfig
//
// When the URI carries a non-zero "otel_config_polling_interval" query
// parameter (e.g. http://localhost:3333/getConfig?otel_config_polling_interval=30s),
// the provider will fetch the configuration on that cadence after the initial
// retrieval and invoke the WatcherFunc supplied to Retrieve when the response
// body changes. The parameter is forwarded to the server unchanged. Without
// the parameter, the provider issues a single GET and never polls, matching
// the historical behavior.
//
// This is used by the http and https external implementations.
func New(scheme SchemeType, set confmap.ProviderSettings) confmap.Provider {
	logger := set.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	return &provider{
		scheme:  scheme,
		logger:  logger,
		cancels: make(map[uint64]context.CancelFunc),
	}
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

func (fmp *provider) Retrieve(ctx context.Context, uri string, watcher confmap.WatcherFunc) (*confmap.Retrieved, error) {
	if !strings.HasPrefix(uri, string(fmp.scheme)+":") {
		return nil, fmt.Errorf("%q uri is not supported by %q provider", uri, string(fmp.scheme))
	}

	if _, err := url.ParseRequestURI(uri); err != nil {
		return nil, fmt.Errorf("invalid uri %q: %w", uri, err)
	}

	// The polling interval is read from, but never removed from, the URI: the
	// parameter is forwarded to the server unchanged (see pollingIntervalQueryParam).
	interval := fmp.pollingIntervalFromURI(uri)

	// Build the HTTP client once and reuse it for the initial fetch and every
	// subsequent poll so connections are kept alive and any TLS material is
	// read from disk a single time instead of on every request.
	client, err := fmp.createClient()
	if err != nil {
		return nil, fmt.Errorf("unable to configure http transport layer: %w", err)
	}

	body, etag, _, err := fmp.fetch(ctx, client, uri, "")
	if err != nil {
		return nil, err
	}

	// Polling is opt-in: only spawn a watcher goroutine when a non-zero
	// interval was supplied AND the caller actually wants to be notified of
	// changes. Either condition missing means we behave exactly like the
	// historical one-shot path.
	if interval == 0 || watcher == nil {
		return confmap.NewRetrievedFromYAML(body)
	}

	pollCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	fmp.mu.Lock()
	id := fmp.nextID
	fmp.nextID++
	fmp.cancels[id] = cancel
	fmp.mu.Unlock()

	fmp.wg.Go(func() {
		fmp.poll(pollCtx, client, uri, etag, sha256.Sum256(body), interval, watcher)
	})

	return confmap.NewRetrievedFromYAML(body, confmap.WithRetrievedClose(func(context.Context) error {
		fmp.mu.Lock()
		delete(fmp.cancels, id)
		fmp.mu.Unlock()
		cancel()
		return nil
	}))
}

func (fmp *provider) Scheme() string {
	return string(fmp.scheme)
}

func (fmp *provider) Shutdown(ctx context.Context) error {
	fmp.mu.Lock()
	for _, c := range fmp.cancels {
		c()
	}
	// Keep the map non-nil (just emptied) so a Retrieve that races in after
	// Shutdown cannot panic assigning into a nil map.
	clear(fmp.cancels)
	fmp.mu.Unlock()

	done := make(chan struct{})
	go func() {
		fmp.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// fetch issues a single conditional GET against fetchURI. When ifNoneMatch is
// non-empty it is forwarded as the "If-None-Match" request header, allowing a
// cooperating server to short-circuit unchanged responses with a 304. The
// returned etag is the value of the response "ETag" header (empty when the
// server does not advertise one), or the supplied ifNoneMatch when the server
// answered with 304.
func (fmp *provider) fetch(ctx context.Context, client *http.Client, fetchURI, ifNoneMatch string) ([]byte, string, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fetchURI, http.NoBody)
	if err != nil {
		return nil, "", 0, fmt.Errorf("unable to construct request for uri %q: %w", fetchURI, err)
	}
	if ifNoneMatch != "" {
		req.Header.Set("If-None-Match", ifNoneMatch)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", 0, fmt.Errorf("unable to download the file via HTTP GET for uri %q: %w ", fetchURI, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		return nil, ifNoneMatch, resp.StatusCode, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, "", resp.StatusCode, fmt.Errorf("failed to load resource from uri %q. status code: %d", fetchURI, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", resp.StatusCode, fmt.Errorf("fail to read the response body from uri %q: %w", fetchURI, err)
	}

	return body, resp.Header.Get("ETag"), resp.StatusCode, nil
}

// poll re-fetches fetchURI on the supplied cadence and invokes watcher exactly
// once when the response body changes, then returns. The collector resolver
// will then close the associated Retrieved (canceling this goroutine if it is
// still running) and call Retrieve again to pick up the new config.
//
// Transport errors are logged at WARN and do not surface as ChangeEvent.Error:
// transient blips would otherwise force the collector into a shutdown for
// every momentary outage, which is worse than serving stale config briefly.
//
// Change detection prefers the server-supplied ETag when available and falls
// back to a SHA-256 of the response body, so the feature also works against
// servers that do not advertise ETags.
func (fmp *provider) poll(
	ctx context.Context,
	client *http.Client,
	fetchURI,
	etag string,
	bodyHash [sha256.Size]byte,
	interval time.Duration,
	watcher confmap.WatcherFunc,
) {
	t := time.NewTicker(interval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}

		body, newEtag, status, err := fmp.fetch(ctx, client, fetchURI, etag)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			fmp.logger.Warn("config polling failed; will retry on next interval",
				zap.String("uri", fetchURI),
				zap.Error(err),
			)
			continue
		}

		if status == http.StatusNotModified {
			continue
		}

		// 200 OK with body. Compare ETag first if both sides supplied one;
		// otherwise compare a body hash so ETag-less servers still work.
		if newEtag != "" && etag != "" && newEtag == etag {
			continue
		}
		newHash := sha256.Sum256(body)
		if newEtag == "" && etag == "" && newHash == bodyHash {
			continue
		}

		fmp.logger.Info("config changed; signaling reload",
			zap.String("uri", fetchURI),
		)
		watcher(&confmap.ChangeEvent{})
		return
	}
}

// pollingIntervalFromURI reads the polling cadence requested by the
// pollingIntervalQueryParam query parameter. A return of 0 means "do not poll",
// which is the behavior for an absent, empty, or explicitly zero value.
//
// The parameter is intentionally left in uri (it is not stripped): callers fetch
// the original URI so the parameter is forwarded to the server unchanged. A
// non-empty value that cannot be interpreted as a non-negative duration is
// logged at WARN and disables polling rather than failing - the value is then
// treated as if it belonged to the upstream server. This keeps the Collector
// from refusing to start when a server happens to use the same parameter name.
func (fmp *provider) pollingIntervalFromURI(uri string) time.Duration {
	if !strings.Contains(uri, pollingIntervalQueryParam) {
		// Hot path: nothing to parse for users who have not opted in.
		return 0
	}

	parsed, err := url.Parse(uri)
	if err != nil {
		// Malformed URI; Retrieve's url.ParseRequestURI surfaces the error.
		return 0
	}

	raw := parsed.Query().Get(pollingIntervalQueryParam)
	if raw == "" {
		return 0
	}

	interval, ok := parseInterval(raw)
	if !ok {
		fmp.logger.Warn(
			"ignoring "+pollingIntervalQueryParam+" query parameter: not a valid non-negative duration; "+
				"treating it as an upstream query parameter and not polling",
			zap.String("value", raw),
		)
		return 0
	}
	return interval
}

// parseInterval interprets a polling-interval value. It accepts any string
// understood by time.ParseDuration (e.g. "30s", "5m") and, as a convenience,
// a bare number with no unit which is taken to mean seconds (e.g. "2" == "2s").
// It returns ok=false for values that are unparseable or negative.
func parseInterval(raw string) (time.Duration, bool) {
	if d, err := time.ParseDuration(raw); err == nil {
		if d < 0 {
			return 0, false
		}
		return d, true
	}
	if n, err := strconv.ParseFloat(raw, 64); err == nil {
		if n < 0 {
			return 0, false
		}
		return time.Duration(n * float64(time.Second)), true
	}
	return 0, false
}
