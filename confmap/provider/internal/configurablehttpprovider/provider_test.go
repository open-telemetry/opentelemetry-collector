// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

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
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/confmaptest"
	"go.opentelemetry.io/collector/internal/testutil"
)

func newConfigurableHTTPProvider(scheme SchemeType, set confmap.ProviderSettings) *provider {
	return New(scheme, set).(*provider)
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
func generateCertificate(t *testing.T, hostname string) (cert, key string, err error) {
	testutil.SkipIfFIPSOnly(t, "x509.CreateCertificate uses SHA-1")
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

	tempDir := t.TempDir()
	certOut, err := os.CreateTemp(tempDir, "cert*.pem")
	if err != nil {
		return "", "", fmt.Errorf("Failed to open cert.pem for writing: %w", err)
	}

	defer certOut.Close()

	if err = pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return "", "", fmt.Errorf("Failed to write data to cert.pem: %w", err)
	}

	keyOut, err := os.CreateTemp(tempDir, "key*.pem")
	if err != nil {
		return "", "", fmt.Errorf("Failed to open key.pem for writing: %w", err)
	}
	defer keyOut.Close()

	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return "", "", fmt.Errorf("Unable to marshal private key: %w", err)
	}

	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes}); err != nil {
		return "", "", fmt.Errorf("Failed to write data to key.pem: %w", err)
	}

	return certOut.Name(), keyOut.Name(), nil
}

func TestFunctionalityDownloadFileHTTP(t *testing.T) {
	fp := newConfigurableHTTPProvider(HTTPScheme, confmaptest.NewNopProviderSettings())
	ts := httptest.NewServer(http.HandlerFunc(answerGet))
	defer ts.Close()
	_, err := fp.Retrieve(context.Background(), ts.URL, nil)
	assert.NoError(t, err)
	assert.NoError(t, fp.Shutdown(context.Background()))
}

func TestFunctionalityDownloadFileHTTPS(t *testing.T) {
	certPath, keyPath, err := generateCertificate(t, "localhost")
	require.NoError(t, err)

	invalidCert, err := os.CreateTemp(t.TempDir(), "cert*.crt")
	defer func() { require.NoError(t, invalidCert.Close()) }()
	require.NoError(t, err)
	_, err = invalidCert.Write([]byte{0, 1, 2})
	require.NoError(t, err)

	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	require.NoError(t, err)
	ts := httptest.NewUnstartedServer(http.HandlerFunc(answerGet))
	ts.TLS = &tls.Config{Certificates: []tls.Certificate{cert}}
	ts.StartTLS()

	defer ts.Close()

	tests := []struct {
		name                   string
		certPath               string
		hostName               string
		useCertificate         bool
		skipHostnameValidation bool
		shouldError            bool
	}{
		{
			name:                   "Test valid certificate and name",
			certPath:               certPath,
			hostName:               "localhost",
			useCertificate:         true,
			skipHostnameValidation: false,
			shouldError:            false,
		},
		{
			name:                   "Test valid certificate with invalid name",
			certPath:               certPath,
			hostName:               "127.0.0.1",
			useCertificate:         true,
			skipHostnameValidation: false,
			shouldError:            true,
		},
		{
			name:                   "Test valid certificate with invalid name, skip validation",
			certPath:               certPath,
			hostName:               "127.0.0.1",
			useCertificate:         true,
			skipHostnameValidation: true,
			shouldError:            false,
		},
		{
			name:                   "Test no certificate should fail",
			certPath:               certPath,
			hostName:               "localhost",
			useCertificate:         false,
			skipHostnameValidation: false,
			shouldError:            true,
		},
		{
			name:                   "Test invalid cert",
			certPath:               invalidCert.Name(),
			hostName:               "localhost",
			useCertificate:         true,
			skipHostnameValidation: false,
			shouldError:            true,
		},
		{
			name:                   "Test no cert",
			certPath:               "no_certificate",
			hostName:               "localhost",
			useCertificate:         true,
			skipHostnameValidation: false,
			shouldError:            true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fp := newConfigurableHTTPProvider(HTTPSScheme, confmaptest.NewNopProviderSettings())
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
	fp := New(HTTPScheme, confmaptest.NewNopProviderSettings())
	_, err := fp.Retrieve(context.Background(), "https://...", nil)
	require.Error(t, err)
	require.NoError(t, fp.Shutdown(context.Background()))

	fp = New(HTTPSScheme, confmaptest.NewNopProviderSettings())
	_, err = fp.Retrieve(context.Background(), "http://...", nil)
	require.Error(t, err)
	assert.NoError(t, fp.Shutdown(context.Background()))
}

func TestEmptyURI(t *testing.T) {
	fp := New(HTTPScheme, confmaptest.NewNopProviderSettings())
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer ts.Close()
	_, err := fp.Retrieve(context.Background(), ts.URL, nil)
	require.Error(t, err)
	require.NoError(t, fp.Shutdown(context.Background()))
}

func TestRetrieveFromShutdownServer(t *testing.T) {
	fp := New(HTTPScheme, confmaptest.NewNopProviderSettings())
	ts := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	ts.Close()
	_, err := fp.Retrieve(context.Background(), ts.URL, nil)
	require.Error(t, err)
	require.NoError(t, fp.Shutdown(context.Background()))
}

func TestNonExistent(t *testing.T) {
	fp := New(HTTPScheme, confmaptest.NewNopProviderSettings())
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()
	_, err := fp.Retrieve(context.Background(), ts.URL, nil)
	require.Error(t, err)
	require.NoError(t, fp.Shutdown(context.Background()))
}

func TestInvalidYAML(t *testing.T) {
	fp := New(HTTPScheme, confmaptest.NewNopProviderSettings())
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("wrong : ["))
		if err != nil {
			fmt.Println("Write failed: ", err)
		}
	}))
	defer ts.Close()
	ret, err := fp.Retrieve(context.Background(), ts.URL, nil)
	require.NoError(t, err)
	raw, err := ret.AsRaw()
	require.NoError(t, err)
	assert.Equal(t, "wrong : [", raw)
	require.NoError(t, fp.Shutdown(context.Background()))
}

func TestScheme(t *testing.T) {
	fp := New(HTTPScheme, confmaptest.NewNopProviderSettings())
	assert.Equal(t, "http", fp.Scheme())
	require.NoError(t, fp.Shutdown(context.Background()))
}

func TestValidateProviderScheme(t *testing.T) {
	assert.NoError(t, confmaptest.ValidateProviderScheme(New(HTTPScheme, confmaptest.NewNopProviderSettings())))
}

func TestInvalidURI(t *testing.T) {
	fp := New(HTTPScheme, confmaptest.NewNopProviderSettings())

	tests := []struct {
		uri string
		err string
	}{
		{
			uri: "foo://..",
			err: "uri is not supported by \"http\" provider",
		},
		{
			uri: "http://",
			err: "no Host in request URL",
		},
		{
			uri: "http://{}",
			err: "invalid character \"{\" in host name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.uri, func(t *testing.T) {
			_, err := fp.Retrieve(context.Background(), tt.uri, nil)
			assert.ErrorContains(t, err, tt.err)
		})
	}
}

const (
	// pollInterval is short enough to keep tests fast but long enough to
	// give a slow CI a fair chance of seeing more than one tick before the
	// test deadline; 25ms is a sweet spot used elsewhere in the repo.
	testPollInterval = 25 * time.Millisecond
	// testWatcherTimeout bounds how long a test will wait for an expected
	// watcher invocation before failing. It must be a healthy multiple of
	// testPollInterval to absorb scheduler jitter.
	testWatcherTimeout = 5 * time.Second
	// testQuietPeriod is how long we wait while asserting that no watcher
	// invocation occurs. It must be at least a few testPollInterval ticks.
	testQuietPeriod = 250 * time.Millisecond
)

// pollingTestServer hosts a configurable response body and ETag that the test
// can mutate at any time, recording every observed request URL so callers can
// assert which query parameters the provider forwards to the server.
type pollingTestServer struct {
	t            *testing.T
	mu           atomic.Pointer[pollingResponse]
	requestCount atomic.Int64
	server       *httptest.Server
	requests     chan *http.Request
}

type pollingResponse struct {
	body       string
	etag       string // when empty, server emits no ETag header
	statusCode int    // 0 means 200 unless body is unset
}

func newPollingTestServer(t *testing.T) *pollingTestServer {
	t.Helper()
	pts := &pollingTestServer{t: t, requests: make(chan *http.Request, 256)}
	pts.setResponse("initial", `"etag-1"`)
	pts.server = httptest.NewServer(http.HandlerFunc(pts.handle))
	t.Cleanup(pts.server.Close)
	return pts
}

func (pts *pollingTestServer) handle(w http.ResponseWriter, r *http.Request) {
	pts.requestCount.Add(1)
	select {
	case pts.requests <- r.Clone(r.Context()):
	default:
	}

	resp := pts.mu.Load()
	if resp == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	if resp.statusCode != 0 && resp.statusCode != http.StatusOK {
		w.WriteHeader(resp.statusCode)
		return
	}

	// Honor If-None-Match / ETag for the cooperating-server case.
	if resp.etag != "" {
		if inm := r.Header.Get("If-None-Match"); inm != "" && inm == resp.etag {
			w.Header().Set("ETag", resp.etag)
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("ETag", resp.etag)
	}

	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprint(w, resp.body)
}

func (pts *pollingTestServer) setResponse(body, etag string) {
	pts.mu.Store(&pollingResponse{body: body, etag: etag})
}

func (pts *pollingTestServer) setStatus(status int) {
	pts.mu.Store(&pollingResponse{statusCode: status})
}

func (pts *pollingTestServer) URL() string { return pts.server.URL }

// retrieveURI builds a URI suitable for fp.Retrieve calls. extra is appended to
// the existing query string; pass "" to leave the URL untouched.
func (pts *pollingTestServer) retrieveURI(extra string) string {
	if extra == "" {
		return pts.URL()
	}
	return pts.URL() + "?" + extra
}

// expectQuiet asserts that no watcher invocation happens within testQuietPeriod.
func expectQuiet(t *testing.T, fired chan struct{}) {
	t.Helper()
	select {
	case <-fired:
		t.Fatal("watcher fired unexpectedly")
	case <-time.After(testQuietPeriod):
	}
}

func waitForWatcher(t *testing.T, fired chan struct{}) {
	t.Helper()
	select {
	case <-fired:
	case <-time.After(testWatcherTimeout):
		t.Fatal("timed out waiting for watcher to fire")
	}
}

// makeWatcher returns a WatcherFunc that closes fired exactly once and counts
// how many times it has been called so the test can assert single-shot.
func makeWatcher() (confmap.WatcherFunc, *atomic.Int32, chan struct{}) {
	fired := make(chan struct{})
	count := &atomic.Int32{}
	w := func(*confmap.ChangeEvent) {
		if count.Add(1) == 1 {
			close(fired)
		}
	}
	return w, count, fired
}

// drainRequests pulls every observed request off the channel for inspection.
func (pts *pollingTestServer) drainRequests() []*http.Request {
	var out []*http.Request
	for {
		select {
		case r := <-pts.requests:
			out = append(out, r)
		default:
			return out
		}
	}
}

func TestPolling_DisabledByDefault(t *testing.T) {
	pts := newPollingTestServer(t)
	fp := newConfigurableHTTPProvider(HTTPScheme, confmaptest.NewNopProviderSettings())
	t.Cleanup(func() { require.NoError(t, fp.Shutdown(context.Background())) })

	watcher, count, fired := makeWatcher()
	ret, err := fp.Retrieve(context.Background(), pts.URL(), watcher)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, ret.Close(context.Background())) })

	// Without otel_config_polling_interval, the provider must not poll. Mutate the
	// server response and assert the watcher stays silent.
	pts.setResponse("changed", `"etag-2"`)
	expectQuiet(t, fired)
	assert.Equal(t, int32(0), count.Load())
	assert.EqualValues(t, 1, pts.requestCount.Load(), "expected exactly one request when polling is disabled")
}

func TestPolling_ZeroIntervalDisabled(t *testing.T) {
	pts := newPollingTestServer(t)
	fp := newConfigurableHTTPProvider(HTTPScheme, confmaptest.NewNopProviderSettings())
	t.Cleanup(func() { require.NoError(t, fp.Shutdown(context.Background())) })

	watcher, count, fired := makeWatcher()
	ret, err := fp.Retrieve(context.Background(), pts.retrieveURI("otel_config_polling_interval=0s"), watcher)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, ret.Close(context.Background())) })

	pts.setResponse("changed", `"etag-2"`)
	expectQuiet(t, fired)
	assert.Equal(t, int32(0), count.Load())
}

func TestPolling_NilWatcherDisablesPolling(t *testing.T) {
	pts := newPollingTestServer(t)
	fp := newConfigurableHTTPProvider(HTTPScheme, confmaptest.NewNopProviderSettings())
	t.Cleanup(func() { require.NoError(t, fp.Shutdown(context.Background())) })

	ret, err := fp.Retrieve(context.Background(), pts.retrieveURI("otel_config_polling_interval=25ms"), nil)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, ret.Close(context.Background())) })

	pts.setResponse("changed", `"etag-2"`)
	time.Sleep(testQuietPeriod)
	// Only the initial Retrieve fetch should have hit the server; without a
	// watcher there is no point polling.
	assert.EqualValues(t, 1, pts.requestCount.Load())
}

func TestPolling_DetectsChangeViaETag(t *testing.T) {
	pts := newPollingTestServer(t)
	fp := newConfigurableHTTPProvider(HTTPScheme, confmaptest.NewNopProviderSettings())
	t.Cleanup(func() { require.NoError(t, fp.Shutdown(context.Background())) })

	watcher, count, fired := makeWatcher()
	uri := pts.retrieveURI("otel_config_polling_interval=" + testPollInterval.String())
	ret, err := fp.Retrieve(context.Background(), uri, watcher)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, ret.Close(context.Background())) })

	pts.setResponse("changed", `"etag-2"`)
	waitForWatcher(t, fired)

	// Watcher must be single-shot per Retrieve; even after another change,
	// the existing goroutine should already have exited.
	pts.setResponse("changed-again", `"etag-3"`)
	time.Sleep(testQuietPeriod)
	assert.Equal(t, int32(1), count.Load())
}

func TestPolling_DetectsChangeViaBodyHashWhenNoETag(t *testing.T) {
	pts := newPollingTestServer(t)
	pts.setResponse("initial", "")
	fp := newConfigurableHTTPProvider(HTTPScheme, confmaptest.NewNopProviderSettings())
	t.Cleanup(func() { require.NoError(t, fp.Shutdown(context.Background())) })

	watcher, count, fired := makeWatcher()
	uri := pts.retrieveURI("otel_config_polling_interval=" + testPollInterval.String())
	ret, err := fp.Retrieve(context.Background(), uri, watcher)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, ret.Close(context.Background())) })

	// Same body, no ETag - watcher must not fire.
	time.Sleep(testQuietPeriod)
	assert.Equal(t, int32(0), count.Load())

	pts.setResponse("changed", "")
	waitForWatcher(t, fired)
	assert.Equal(t, int32(1), count.Load())
}

func TestPolling_NoChangeStays304(t *testing.T) {
	pts := newPollingTestServer(t)
	fp := newConfigurableHTTPProvider(HTTPScheme, confmaptest.NewNopProviderSettings())
	t.Cleanup(func() { require.NoError(t, fp.Shutdown(context.Background())) })

	watcher, count, fired := makeWatcher()
	uri := pts.retrieveURI("otel_config_polling_interval=" + testPollInterval.String())
	ret, err := fp.Retrieve(context.Background(), uri, watcher)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, ret.Close(context.Background())) })

	// Body and etag never change - watcher must remain silent.
	expectQuiet(t, fired)
	assert.Equal(t, int32(0), count.Load())
	// And we should have observed multiple requests (initial + several polls).
	assert.Greater(t, pts.requestCount.Load(), int64(1), "expected multiple polls to have happened")
}

func TestPolling_TransportErrorsDoNotTearDown(t *testing.T) {
	pts := newPollingTestServer(t)
	fp := newConfigurableHTTPProvider(HTTPScheme, confmaptest.NewNopProviderSettings())
	t.Cleanup(func() { require.NoError(t, fp.Shutdown(context.Background())) })

	watcher, count, fired := makeWatcher()
	uri := pts.retrieveURI("otel_config_polling_interval=" + testPollInterval.String())
	ret, err := fp.Retrieve(context.Background(), uri, watcher)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, ret.Close(context.Background())) })

	// Force the server into a 5xx for a while.
	pts.setStatus(http.StatusInternalServerError)
	time.Sleep(testQuietPeriod)
	assert.Equal(t, int32(0), count.Load(), "watcher must not fire on server-side errors")

	// Recover with a real change; watcher should fire on the first 200 OK.
	pts.setResponse("changed", `"etag-2"`)
	waitForWatcher(t, fired)
	assert.Equal(t, int32(1), count.Load())
}

func TestPolling_ForwardsParamToServer(t *testing.T) {
	pts := newPollingTestServer(t)
	fp := newConfigurableHTTPProvider(HTTPScheme, confmaptest.NewNopProviderSettings())
	t.Cleanup(func() { require.NoError(t, fp.Shutdown(context.Background())) })

	uri := pts.retrieveURI("foo=bar&otel_config_polling_interval=" + testPollInterval.String() + "&baz=qux")
	ret, err := fp.Retrieve(context.Background(), uri, nil)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, ret.Close(context.Background())) })

	requests := pts.drainRequests()
	require.NotEmpty(t, requests)
	for _, r := range requests {
		q := r.URL.Query()
		// The parameter is namespaced and forwarded unchanged so a server that
		// happens to use the same key keeps working - it must NOT be stripped.
		assert.Equal(t, testPollInterval.String(), q.Get("otel_config_polling_interval"),
			"otel_config_polling_interval must be forwarded to the server, not stripped")
		assert.Equal(t, "bar", q.Get("foo"), "unrelated query parameters must be preserved")
		assert.Equal(t, "qux", q.Get("baz"))
	}
}

func TestPolling_MalformedURIReturnsError(t *testing.T) {
	fp := newConfigurableHTTPProvider(HTTPScheme, confmaptest.NewNopProviderSettings())
	t.Cleanup(func() { require.NoError(t, fp.Shutdown(context.Background())) })

	// %zz is an invalid percent-encoding sequence, so url.ParseRequestURI in
	// Retrieve rejects the URI before any fetch is attempted - independent of
	// the polling parameter.
	_, err := fp.Retrieve(context.Background(), "http://example.com/%zz?otel_config_polling_interval=1s", nil)
	require.Error(t, err)
	assert.ErrorContains(t, err, "invalid uri")
}

func TestParseInterval(t *testing.T) {
	tests := []struct {
		raw  string
		want time.Duration
		ok   bool
	}{
		{"30s", 30 * time.Second, true},
		{"5m", 5 * time.Minute, true},
		{"100ms", 100 * time.Millisecond, true},
		// A bare number with no unit is interpreted as seconds.
		{"30", 30 * time.Second, true},
		{"2", 2 * time.Second, true},
		{"1.5", 1500 * time.Millisecond, true},
		// Zero is valid and means "do not poll".
		{"0", 0, true},
		{"0s", 0, true},
		// Unparseable or negative values are rejected.
		{"not-a-duration", 0, false},
		{"-1s", 0, false},
		{"-2", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			got, ok := parseInterval(tt.raw)
			assert.Equal(t, tt.ok, ok)
			if tt.ok {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestPolling_BareNumberEnablesPolling(t *testing.T) {
	// A bare number with no unit is interpreted as seconds, so "1" enables
	// polling on a one-second cadence.
	pts := newPollingTestServer(t)
	fp := newConfigurableHTTPProvider(HTTPScheme, confmaptest.NewNopProviderSettings())
	t.Cleanup(func() { require.NoError(t, fp.Shutdown(context.Background())) })

	watcher, count, fired := makeWatcher()
	ret, err := fp.Retrieve(context.Background(), pts.retrieveURI("otel_config_polling_interval=1"), watcher)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, ret.Close(context.Background())) })

	pts.setResponse("changed", `"etag-2"`)
	waitForWatcher(t, fired)
	assert.Equal(t, int32(1), count.Load())
}

func TestPolling_InvalidValueWarnsAndDisablesPolling(t *testing.T) {
	// A non-empty value that is not a valid non-negative duration must not fail
	// Retrieve: it is treated as if it belonged to the upstream server (polling
	// stays off) and the operator gets a WARN so a fat-fingered real interval
	// is still noticeable.
	for _, val := range []string{"not-a-duration", "-1s", "-2"} {
		t.Run(val, func(t *testing.T) {
			pts := newPollingTestServer(t)
			core, logs := observer.New(zapcore.WarnLevel)
			fp := newConfigurableHTTPProvider(HTTPScheme, confmap.ProviderSettings{Logger: zap.New(core)})
			t.Cleanup(func() { require.NoError(t, fp.Shutdown(context.Background())) })

			watcher, count, fired := makeWatcher()
			ret, err := fp.Retrieve(context.Background(), pts.retrieveURI("otel_config_polling_interval="+val), watcher)
			require.NoError(t, err, "an invalid interval must not fail Retrieve")
			t.Cleanup(func() { require.NoError(t, ret.Close(context.Background())) })

			pts.setResponse("changed", `"etag-2"`)
			expectQuiet(t, fired)
			assert.Equal(t, int32(0), count.Load())
			assert.EqualValues(t, 1, pts.requestCount.Load(),
				"invalid interval must disable polling (only the initial fetch)")
			assert.NotEmpty(t, logs.FilterMessageSnippet("otel_config_polling_interval").All(),
				"expected a WARN about the ignored interval")
		})
	}
}

func TestPolling_ReusesHTTPClientConnection(t *testing.T) {
	// Reusing one *http.Client across the initial fetch and every poll keeps
	// the TCP connection alive, so the server should observe exactly one new
	// connection no matter how many polls happen.
	var (
		newConns atomic.Int64
		reqCount atomic.Int64
	)
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reqCount.Add(1)
		// Stable body and ETag so polling continues forever without firing the
		// watcher; we only care about connection reuse here.
		w.Header().Set("ETag", `"etag-stable"`)
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "same-body")
	}))
	srv.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			newConns.Add(1)
		}
	}
	srv.Start()
	t.Cleanup(srv.Close)

	fp := newConfigurableHTTPProvider(HTTPScheme, confmaptest.NewNopProviderSettings())
	t.Cleanup(func() { require.NoError(t, fp.Shutdown(context.Background())) })

	watcher, _, _ := makeWatcher()
	ret, err := fp.Retrieve(context.Background(), srv.URL+"?otel_config_polling_interval="+testPollInterval.String(), watcher)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, ret.Close(context.Background())) })

	require.Eventually(t, func() bool { return reqCount.Load() >= 4 },
		testWatcherTimeout, testPollInterval, "expected several polls to occur")

	assert.EqualValues(t, 1, newConns.Load(),
		"the provider must reuse a single connection across the initial fetch and all polls")
}

func TestPolling_RetrievedCloseCancelsGoroutine(t *testing.T) {
	pts := newPollingTestServer(t)
	fp := newConfigurableHTTPProvider(HTTPScheme, confmaptest.NewNopProviderSettings())
	t.Cleanup(func() { require.NoError(t, fp.Shutdown(context.Background())) })

	watcher, count, _ := makeWatcher()
	uri := pts.retrieveURI("otel_config_polling_interval=" + testPollInterval.String())
	ret, err := fp.Retrieve(context.Background(), uri, watcher)
	require.NoError(t, err)

	// Close BEFORE making any change. After Close returns, no further
	// requests should be made and the watcher must never fire.
	require.NoError(t, ret.Close(context.Background()))
	preCloseRequests := pts.requestCount.Load()

	pts.setResponse("changed", `"etag-2"`)
	time.Sleep(testQuietPeriod)
	postCloseRequests := pts.requestCount.Load()

	// Allow at most one in-flight request that started before Close took effect.
	assert.LessOrEqual(t, postCloseRequests-preCloseRequests, int64(1),
		"polling continued after Retrieved.Close")
	assert.Equal(t, int32(0), count.Load(), "watcher fired after Retrieved.Close")
}

func TestPolling_ShutdownCancelsActiveGoroutine(t *testing.T) {
	pts := newPollingTestServer(t)
	fp := newConfigurableHTTPProvider(HTTPScheme, confmaptest.NewNopProviderSettings())

	watcher, count, _ := makeWatcher()
	uri := pts.retrieveURI("otel_config_polling_interval=" + testPollInterval.String())
	ret, err := fp.Retrieve(context.Background(), uri, watcher)
	require.NoError(t, err)

	// Confirm the polling goroutine is alive and ticking before Shutdown.
	require.Eventually(t, func() bool { return pts.requestCount.Load() > 1 },
		time.Second, testPollInterval, "polling goroutine never made a follow-up request")

	require.NoError(t, fp.Shutdown(context.Background()))
	preShutdownRequests := pts.requestCount.Load()

	pts.setResponse("changed", `"etag-2"`)
	time.Sleep(testQuietPeriod)
	assert.LessOrEqual(t, pts.requestCount.Load()-preShutdownRequests, int64(1),
		"polling continued after Shutdown")
	assert.Equal(t, int32(0), count.Load(), "watcher fired after Shutdown")

	// Closing the Retrieved after Shutdown must remain a no-op (no panic, no error).
	assert.NoError(t, ret.Close(context.Background()))
	// And Shutdown is idempotent.
	assert.NoError(t, fp.Shutdown(context.Background()))
}

func TestPolling_MultipleRetrieveLifecyclesShareProviderCleanly(t *testing.T) {
	// The confmap.Resolver re-calls Retrieve after each watcher invocation,
	// so a single Provider instance must tolerate many overlapping Retrieve
	// lifecycles. Walk a quick add → close → re-add cycle and confirm
	// Shutdown still drains everything cleanly (goleak in TestMain enforces
	// "everything").
	pts := newPollingTestServer(t)
	fp := newConfigurableHTTPProvider(HTTPScheme, confmaptest.NewNopProviderSettings())
	t.Cleanup(func() { require.NoError(t, fp.Shutdown(context.Background())) })

	uri := pts.retrieveURI("otel_config_polling_interval=" + testPollInterval.String())

	cancelsLen := func() int {
		fp.mu.Lock()
		defer fp.mu.Unlock()
		return len(fp.cancels)
	}

	for range 3 {
		watcher, _, _ := makeWatcher()
		ret, err := fp.Retrieve(context.Background(), uri, watcher)
		require.NoError(t, err)
		assert.Equal(t, 1, cancelsLen(), "an active polling Retrieve must track exactly one cancel")

		require.NoError(t, ret.Close(context.Background()))
		// The cancel must be removed on Close, not just at Shutdown, so the map
		// stays bounded across the many reload cycles a long-running Collector
		// performs.
		assert.Equal(t, 0, cancelsLen(), "Retrieved.Close must remove its own cancel from the map")
	}
}

func TestNew_NilLoggerDefaultsToNop(t *testing.T) {
	// confmaptest.NewNopProviderSettings() always populates Logger, so the
	// nil-logger fallback in New is only reachable when a caller passes
	// confmap.ProviderSettings{} directly. Exercise that path and then run
	// a real Retrieve to make sure the provider is fully usable without a
	// caller-supplied logger.
	pts := newPollingTestServer(t)

	fp := newConfigurableHTTPProvider(HTTPScheme, confmap.ProviderSettings{})
	t.Cleanup(func() { require.NoError(t, fp.Shutdown(context.Background())) })

	require.NotNil(t, fp.logger, "New must default to a non-nil logger")

	ret, err := fp.Retrieve(context.Background(), pts.URL(), nil)
	require.NoError(t, err)
	require.NoError(t, ret.Close(context.Background()))
}

func TestShutdown_CallerContextDeadlineWins(t *testing.T) {
	// Shutdown's `select { case <-done: ; case <-ctx.Done(): }` only
	// returns ctx.Err() when wg.Wait outlives the caller's context. To
	// exercise that path deterministically (no HTTP timing, no ticker
	// jitter), park a goroutine in fp.wg directly and then call Shutdown
	// with a very short deadline.
	fp := newConfigurableHTTPProvider(HTTPScheme, confmaptest.NewNopProviderSettings())

	release := make(chan struct{})
	// sync.OnceFunc lets us (a) close release as cleanup if the test fails
	// early via require, draining the parked goroutine for goleak, and (b)
	// also close it explicitly mid-test to set up the second Shutdown,
	// without risking a double-close panic.
	closeRelease := sync.OnceFunc(func() { close(release) })
	t.Cleanup(closeRelease)
	fp.wg.Go(func() { <-release })

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	err := fp.Shutdown(shutdownCtx)
	require.ErrorIs(t, err, context.DeadlineExceeded)

	// Release the parked goroutine and verify a fresh Shutdown observes a
	// clean drain.
	closeRelease()
	require.NoError(t, fp.Shutdown(context.Background()))
}

func TestRetrieve_NilContextSurfacesNewRequestError(t *testing.T) {
	// http.NewRequestWithContext returns "net/http: nil Context" when its
	// ctx is nil. fetch wraps that error with "unable to construct request"
	// so the caller sees a meaningful message rather than a panic.
	pts := newPollingTestServer(t)
	fp := newConfigurableHTTPProvider(HTTPScheme, confmaptest.NewNopProviderSettings())
	t.Cleanup(func() { require.NoError(t, fp.Shutdown(context.Background())) })

	//nolint:staticcheck // intentionally exercising the defensive nil-context branch in fetch
	_, err := fp.Retrieve(nil, pts.URL(), nil)
	require.Error(t, err)
	assert.ErrorContains(t, err, "unable to construct request")
}

func TestRetrieve_BodyReadFailureSurfacesAsError(t *testing.T) {
	// Build a server that hijacks the connection, advertises a
	// Content-Length of 1024, and then closes without sending any of those
	// bytes. The Go HTTP client accepts the headers and io.ReadAll on the
	// body returns io.ErrUnexpectedEOF. fetch wraps that as "fail to read
	// the response body".
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// testifylint's go-require rule forbids require.* inside an http
		// handler (FailNow only kills the handler goroutine, leaving the
		// test goroutine to wedge). httptest.NewServer's response writer
		// is always a Hijacker and these writes are synchronous to a
		// loopback socket - on the unlikely path that any of them errors,
		// just return so the client sees an empty body and the test's
		// downstream assertion fails with a meaningful message.
		hj, ok := w.(http.Hijacker)
		if !ok {
			return
		}
		conn, bw, err := hj.Hijack()
		if err != nil {
			return
		}
		defer conn.Close()
		if _, err := bw.WriteString("HTTP/1.1 200 OK\r\nContent-Length: 1024\r\n\r\n"); err != nil {
			return
		}
		_ = bw.Flush()
		// Close immediately, well before the promised 1024 bytes arrive.
	}))
	t.Cleanup(srv.Close)

	fp := newConfigurableHTTPProvider(HTTPScheme, confmaptest.NewNopProviderSettings())
	t.Cleanup(func() { require.NoError(t, fp.Shutdown(context.Background())) })

	_, err := fp.Retrieve(context.Background(), srv.URL, nil)
	require.Error(t, err)
	assert.ErrorContains(t, err, "fail to read the response body")
}

func TestPolling_SameETagOn200StaysQuiet(t *testing.T) {
	// Some servers do not honor If-None-Match and instead always return a
	// fresh 200 OK with the same ETag. The polling loop must treat an
	// equal ETag pair as "no change" without consulting body bytes - this
	// hits the `newEtag == etag` continue branch on line 260 and closes
	// the matching && partials around it.
	var (
		mu       atomic.Pointer[pollingResponse]
		reqCount atomic.Int64
	)
	mu.Store(&pollingResponse{body: "initial", etag: `"etag-stable"`})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reqCount.Add(1)
		resp := mu.Load()
		// Deliberately ignore If-None-Match: every poll gets a 200.
		w.Header().Set("ETag", resp.etag)
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, resp.body)
	}))
	t.Cleanup(srv.Close)

	fp := newConfigurableHTTPProvider(HTTPScheme, confmaptest.NewNopProviderSettings())
	t.Cleanup(func() { require.NoError(t, fp.Shutdown(context.Background())) })

	watcher, count, fired := makeWatcher()
	uri := srv.URL + "?otel_config_polling_interval=" + testPollInterval.String()
	ret, err := fp.Retrieve(context.Background(), uri, watcher)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, ret.Close(context.Background())) })

	expectQuiet(t, fired)
	assert.Equal(t, int32(0), count.Load())
	assert.Greater(t, reqCount.Load(), int64(1),
		"polling goroutine should issue follow-up 200s even though ETag is stable")

	// Now flip the ETag - the watcher must fire on the very next poll.
	mu.Store(&pollingResponse{body: "initial", etag: `"etag-rotated"`})
	waitForWatcher(t, fired)
	assert.Equal(t, int32(1), count.Load())
}

func TestPolling_EmptyIntervalParamDisablesPolling(t *testing.T) {
	// `?otel_config_polling_interval=` (key present, value empty) must behave exactly
	// like the missing-parameter case: no polling, no error. This covers
	// the `if raw == ""` early-return branch in pollingIntervalFromURI.
	pts := newPollingTestServer(t)
	fp := newConfigurableHTTPProvider(HTTPScheme, confmaptest.NewNopProviderSettings())
	t.Cleanup(func() { require.NoError(t, fp.Shutdown(context.Background())) })

	watcher, count, fired := makeWatcher()
	ret, err := fp.Retrieve(context.Background(), pts.retrieveURI("otel_config_polling_interval="), watcher)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, ret.Close(context.Background())) })

	pts.setResponse("changed", `"etag-2"`)
	expectQuiet(t, fired)
	assert.Equal(t, int32(0), count.Load())
	assert.EqualValues(t, 1, pts.requestCount.Load(),
		"empty otel_config_polling_interval value must disable polling, like an absent parameter")
}
