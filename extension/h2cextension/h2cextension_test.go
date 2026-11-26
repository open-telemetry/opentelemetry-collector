// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package h2cextension

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/client"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configauth"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configmiddleware"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/extension/extensionauth"
	"go.opentelemetry.io/collector/extension/extensionmiddleware"
)

var (
	h2cID     = component.MustNewID("h2c")
	authID    = component.MustNewID("testauth")
	firstID   = component.MustNewID("first")
	secondID  = component.MustNewID("second")
	readingID = component.MustNewID("reading")
)

func TestGetHTTPServerProtocols(t *testing.T) {
	protocols, err := (&h2cExtension{}).GetHTTPServerProtocols(t.Context())
	require.NoError(t, err)
	require.True(t, protocols.UnencryptedHTTP2())
}

func TestConfighttpServesHTTP1AndPriorKnowledge(t *testing.T) {
	t.Parallel()

	srv := startConfighttpServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), nil)

	for name, httpClient := range map[string]*http.Client{"http1": http1Client(t), "h2c": priorKnowledgeClient(t)} {
		resp, err := httpClient.Get(srv.url)
		require.NoError(t, err, name)
		require.NoError(t, resp.Body.Close())
		require.Equal(t, http.StatusNoContent, resp.StatusCode, name)
		if name == "h2c" {
			require.Equal(t, 2, resp.ProtoMajor)
		} else {
			require.Equal(t, 1, resp.ProtoMajor)
		}
	}
}

func TestConfighttpAuthenticatesEachStream(t *testing.T) {
	t.Parallel()

	subjects := make(chan string, 5)
	srv := startConfighttpServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		subject, _ := client.FromContext(r.Context()).Auth.GetAttribute("subject").(string)
		subjects <- subject
		w.WriteHeader(http.StatusNoContent)
	}), withAuth)

	httpClient := priorKnowledgeClient(t)
	for _, tc := range []struct {
		authorization string
		status        int
		subject       string
	}{
		{authorization: "Bearer alice", status: http.StatusNoContent, subject: "alice"},
		{status: http.StatusUnauthorized},
		{authorization: "Bearer bob", status: http.StatusNoContent, subject: "bob"},
		{authorization: "Bearer ", status: http.StatusUnauthorized},
		{authorization: "Bearer alice", status: http.StatusNoContent, subject: "alice"},
	} {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.url, http.NoBody)
		require.NoError(t, err)
		if tc.authorization != "" {
			req.Header.Set("Authorization", tc.authorization)
		}
		resp, err := httpClient.Do(req)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
		require.Equal(t, 2, resp.ProtoMajor)
		require.Equal(t, tc.status, resp.StatusCode, tc.authorization)
		if tc.subject != "" {
			require.Equal(t, tc.subject, <-subjects)
		}
	}
	require.Empty(t, subjects, "rejected streams must not reach the receiver")
	require.Equal(t, int64(1), srv.connections.Load(), "all streams must share one h2c connection")
}

func TestConfighttpPriorKnowledgeStreamsHaveIndependentClientInfo(t *testing.T) {
	t.Parallel()

	type result struct {
		auth    client.AuthData
		hasAddr bool
	}
	results := make(chan result, 2)
	srv := startConfighttpServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		info := client.FromContext(r.Context())
		results <- result{auth: info.Auth, hasAddr: info.Addr != nil}
		w.WriteHeader(http.StatusNoContent)
	}), nil)

	httpClient := priorKnowledgeClient(t)
	for _, path := range []string{"/first", "/second"} {
		resp, err := httpClient.Get(srv.url + path)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
		require.Equal(t, http.StatusNoContent, resp.StatusCode)
		require.Equal(t, 2, resp.ProtoMajor)
		got := <-results
		require.Nil(t, got.auth)
		require.True(t, got.hasAddr, "client info must be populated for every stream")
	}
}

func TestConfighttpIncludesMetadataPerStream(t *testing.T) {
	t.Parallel()

	tenants := make(chan []string, 3)
	srv := startConfighttpServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenants <- client.FromContext(r.Context()).Metadata.Get("x-tenant")
		w.WriteHeader(http.StatusNoContent)
	}), func(cfg *confighttp.ServerConfig) {
		cfg.IncludeMetadata = true
	})

	httpClient := priorKnowledgeClient(t)
	for _, tenant := range []string{"alpha", "", "beta"} {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.url, http.NoBody)
		require.NoError(t, err)
		if tenant != "" {
			req.Header.Set("X-Tenant", tenant)
		}
		resp, err := httpClient.Do(req)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
		require.Equal(t, 2, resp.ProtoMajor)
		require.Equal(t, http.StatusNoContent, resp.StatusCode)

		got := <-tenants
		if tenant == "" {
			require.Empty(t, got, "a stream must not see metadata from earlier streams")
			continue
		}
		require.Equal(t, []string{tenant}, got)
	}
	require.Equal(t, int64(1), srv.connections.Load(), "all streams must share one h2c connection")
}

func TestConfighttpRecordsSpanPerStream(t *testing.T) {
	t.Parallel()

	tel := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tel.Shutdown(context.Background())) })
	srv := startConfighttpServerWithSettings(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), nil, tel.NewTelemetrySettings())

	const streams = 3
	httpClient := priorKnowledgeClient(t)
	for range streams {
		resp, err := httpClient.Get(srv.url)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
		require.Equal(t, 2, resp.ProtoMajor)
		require.Equal(t, http.StatusNoContent, resp.StatusCode)
	}
	require.Equal(t, int64(1), srv.connections.Load(), "all streams must share one h2c connection")

	require.Eventually(t, func() bool { return len(tel.SpanRecorder.Ended()) == streams }, 5*time.Second, 10*time.Millisecond)
	traceIDs := map[trace.TraceID]struct{}{}
	for _, span := range tel.SpanRecorder.Ended() {
		require.Equal(t, trace.SpanKindServer, span.SpanKind())
		require.False(t, span.Parent().IsValid(), "a stream span must not be parented by another stream")
		require.Contains(t, span.Attributes(), attribute.String("network.protocol.version", "2.0"))
		traceIDs[span.SpanContext().TraceID()] = struct{}{}
	}
	require.Len(t, traceIDs, streams, "each stream must start its own trace")
}

func TestConfighttpAppliesCORSAndResponseHeadersPerStream(t *testing.T) {
	t.Parallel()

	const allowed = "https://allowed.example"
	srv := startConfighttpServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), func(cfg *confighttp.ServerConfig) {
		cfg.CORS = configoptional.Some(confighttp.CORSConfig{AllowedOrigins: []string{allowed}})
		cfg.ResponseHeaders = configopaque.MapList{{Name: "X-Served-By", Value: "collector"}}
	})

	httpClient := priorKnowledgeClient(t)
	for _, tc := range []struct {
		name        string
		method      string
		origin      string
		allowOrigin string
	}{
		{name: "preflight", method: http.MethodOptions, origin: allowed, allowOrigin: allowed},
		{name: "allowed_origin", method: http.MethodPost, origin: allowed, allowOrigin: allowed},
		{name: "other_origin", method: http.MethodPost, origin: "https://other.example"},
	} {
		req, err := http.NewRequestWithContext(t.Context(), tc.method, srv.url, http.NoBody)
		require.NoError(t, err)
		req.Header.Set("Origin", tc.origin)
		if tc.method == http.MethodOptions {
			req.Header.Set("Access-Control-Request-Method", http.MethodPost)
		}
		resp, err := httpClient.Do(req)
		require.NoError(t, err, tc.name)
		require.NoError(t, resp.Body.Close())
		require.Equal(t, 2, resp.ProtoMajor, tc.name)
		require.Equal(t, http.StatusNoContent, resp.StatusCode, tc.name)
		require.Equal(t, tc.allowOrigin, resp.Header.Get("Access-Control-Allow-Origin"), tc.name)
		require.Equal(t, "collector", resp.Header.Get("X-Served-By"), tc.name)
	}
	require.Equal(t, int64(1), srv.connections.Load(), "all streams must share one h2c connection")
}

func TestConfighttpLimitsRequestBodySizePerStream(t *testing.T) {
	t.Parallel()

	const limit = 1024
	srv := startConfighttpServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.ReadAll(r.Body); err != nil {
			if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
				w.WriteHeader(http.StatusRequestEntityTooLarge)
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}), func(cfg *confighttp.ServerConfig) {
		cfg.MaxRequestBodySize = limit
	})

	var compressed bytes.Buffer
	gz := gzip.NewWriter(&compressed)
	_, err := gz.Write(make([]byte, 2*limit))
	require.NoError(t, err)
	require.NoError(t, gz.Close())
	require.Less(t, compressed.Len(), limit, "the compressed body must pass the raw limit")

	httpClient := priorKnowledgeClient(t)
	for _, tc := range []struct {
		name     string
		body     []byte
		encoding string
		status   int
	}{
		{name: "at_limit", body: make([]byte, limit), status: http.StatusNoContent},
		{name: "over_limit", body: make([]byte, limit+1), status: http.StatusRequestEntityTooLarge},
		{name: "decompressed_over_limit", body: compressed.Bytes(), encoding: "gzip", status: http.StatusRequestEntityTooLarge},
	} {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, srv.url, bytes.NewReader(tc.body))
		require.NoError(t, err)
		if tc.encoding != "" {
			req.Header.Set("Content-Encoding", tc.encoding)
		}
		resp, err := httpClient.Do(req)
		require.NoError(t, err, tc.name)
		require.NoError(t, resp.Body.Close())
		require.Equal(t, 2, resp.ProtoMajor, tc.name)
		require.Equal(t, tc.status, resp.StatusCode, tc.name)
	}
	require.Equal(t, int64(1), srv.connections.Load(), "all streams must share one h2c connection")
}

// Middlewares read the same size-limited, decoded body the receiver does.
func TestConfighttpMiddlewareReadsLimitedBody(t *testing.T) {
	t.Parallel()

	const limit = 1024
	var reached atomic.Int64
	srv := startConfighttpServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reached.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}), func(cfg *confighttp.ServerConfig) {
		cfg.MaxRequestBodySize = limit
		cfg.Middlewares = append(cfg.Middlewares, configmiddleware.Config{ID: readingID})
	})

	var compressed bytes.Buffer
	gz := gzip.NewWriter(&compressed)
	_, err := gz.Write(make([]byte, 2*limit))
	require.NoError(t, err)
	require.NoError(t, gz.Close())
	require.Less(t, compressed.Len(), limit, "the compressed body must pass the raw limit")

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	httpClient := priorKnowledgeClient(t)
	for _, tc := range []struct {
		name     string
		body     []byte
		encoding string
		status   int
	}{
		{name: "at_limit", body: make([]byte, limit), status: http.StatusNoContent},
		{name: "over_limit", body: make([]byte, limit+1), status: http.StatusRequestEntityTooLarge},
		{name: "decompressed_over_limit", body: compressed.Bytes(), encoding: "gzip", status: http.StatusRequestEntityTooLarge},
	} {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, srv.url, bytes.NewReader(tc.body))
		require.NoError(t, err)
		if tc.encoding != "" {
			req.Header.Set("Content-Encoding", tc.encoding)
		}
		resp, err := httpClient.Do(req)
		require.NoError(t, err, tc.name)
		require.NoError(t, resp.Body.Close())
		require.Equal(t, 2, resp.ProtoMajor, tc.name)
		require.Equal(t, tc.status, resp.StatusCode, tc.name)
		limited := tc.status == http.StatusRequestEntityTooLarge
		require.Equal(t, limited, resp.Header.Get("X-Middleware-Limited") == "true", "%s: the middleware must hit the limit while reading", tc.name)
	}
	require.Equal(t, int64(1), reached.Load(), "only the request within the limit may reach the receiver")
}

func TestConfighttpDecompressesEachStream(t *testing.T) {
	t.Parallel()

	type received struct {
		encoding string
		body     []byte
	}
	bodies := make(chan received, 2)
	srv := startConfighttpServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		bodies <- received{encoding: r.Header.Get("Content-Encoding"), body: body}
		w.WriteHeader(http.StatusNoContent)
	}), nil)

	payload := []byte(strings.Repeat("otlp payload ", 64))
	var compressed bytes.Buffer
	gz := gzip.NewWriter(&compressed)
	_, err := gz.Write(payload)
	require.NoError(t, err)
	require.NoError(t, gz.Close())

	httpClient := priorKnowledgeClient(t)
	for _, encoding := range []string{"gzip", ""} {
		body := payload
		if encoding != "" {
			body = compressed.Bytes()
		}
		req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, srv.url, bytes.NewReader(body))
		require.NoError(t, err)
		if encoding != "" {
			req.Header.Set("Content-Encoding", encoding)
		}
		resp, err := httpClient.Do(req)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
		require.Equal(t, 2, resp.ProtoMajor)
		require.Equal(t, http.StatusNoContent, resp.StatusCode)

		got := <-bodies
		require.Empty(t, got.encoding, "the handler must see a decoded body")
		require.Equal(t, payload, got.body)
	}
	require.Equal(t, int64(1), srv.connections.Load(), "all streams must share one h2c connection")
}

func TestConfighttpShutdownDrainsStreams(t *testing.T) {
	t.Parallel()

	var entries atomic.Int64
	started := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseHandler := func() { releaseOnce.Do(func() { close(release) }) }
	t.Cleanup(releaseHandler)
	srv := startConfighttpServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if entries.Add(1) == 1 {
			close(started)
		}
		<-release
		w.WriteHeader(http.StatusNoContent)
	}), nil)

	httpClient := priorKnowledgeClient(t)
	inFlight := make(chan *http.Response, 1)
	go func() {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.url, http.NoBody)
		if err != nil {
			close(inFlight)
			return
		}
		resp, err := httpClient.Do(req)
		if err != nil {
			close(inFlight)
			return
		}
		_ = resp.Body.Close()
		inFlight <- resp
	}()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "the first stream never reached the handler")
	}

	shutdownCtx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	shutdown := make(chan error, 1)
	go func() { shutdown <- srv.server.Shutdown(shutdownCtx) }()

	require.Never(t, func() bool { return len(shutdown) > 0 }, 200*time.Millisecond, 10*time.Millisecond,
		"shutdown must wait for the in-flight stream")
	// The client may see GOAWAY and fail the request without sending it, so
	// this proves that requests fail and never run, not how the server
	// treats a late stream. The timeout keeps a regression from hanging.
	lateCtx, lateCancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer lateCancel()
	req, err := http.NewRequestWithContext(lateCtx, http.MethodGet, srv.url, http.NoBody)
	require.NoError(t, err)
	resp, err := httpClient.Do(req)
	if err == nil {
		require.NoError(t, resp.Body.Close())
	}
	require.Error(t, err, "new requests must fail after shutdown starts")
	require.Equal(t, int64(1), entries.Load(), "a new request reached the handler after shutdown started")
	require.NotErrorIs(t, err, context.DeadlineExceeded, "the late request must fail because of shutdown, not the timeout")

	releaseHandler()
	var ok bool
	select {
	case resp, ok = <-inFlight:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "the in-flight stream never completed")
	}
	require.True(t, ok, "the in-flight stream must complete")
	require.Equal(t, 2, resp.ProtoMajor)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	require.NoError(t, <-shutdown, "shutdown must close the h2c connection")
	<-srv.serveDone
	require.Equal(t, int64(1), srv.connections.Load())
}

// h2c only changes the server's protocols, so its position in the list does
// not change the request path. Other middlewares keep their configured order
// and run after authentication and decompression.
func TestConfighttpMiddlewareOrderIgnoresH2CPosition(t *testing.T) {
	t.Parallel()

	payload := []byte(strings.Repeat("otlp payload ", 64))
	var compressed bytes.Buffer
	gz := gzip.NewWriter(&compressed)
	_, err := gz.Write(payload)
	require.NoError(t, err)
	require.NoError(t, gz.Close())

	for name, ids := range map[string][]component.ID{
		"h2c_first":  {h2cID, firstID, secondID},
		"h2c_middle": {firstID, h2cID, secondID},
		"h2c_last":   {firstID, secondID, h2cID},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			saw := make(chan []string, 2)
			srv := startConfighttpServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				if err != nil || !bytes.Equal(payload, body) {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				saw <- r.Header.Values("X-Middleware-Saw")
				w.WriteHeader(http.StatusNoContent)
			}), func(cfg *confighttp.ServerConfig) {
				withAuth(cfg)
				cfg.Middlewares = nil
				for _, id := range ids {
					cfg.Middlewares = append(cfg.Middlewares, configmiddleware.Config{ID: id})
				}
			})

			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			httpClient := priorKnowledgeClient(t)
			for _, authorization := range []string{"", "Bearer alice"} {
				req, err := http.NewRequestWithContext(ctx, http.MethodPost, srv.url, bytes.NewReader(compressed.Bytes()))
				require.NoError(t, err)
				req.Header.Set("Content-Encoding", "gzip")
				if authorization != "" {
					req.Header.Set("Authorization", authorization)
				}
				resp, err := httpClient.Do(req)
				require.NoError(t, err)
				require.NoError(t, resp.Body.Close())
				require.Equal(t, 2, resp.ProtoMajor)

				if authorization == "" {
					require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
					require.Empty(t, resp.Header.Values("X-Middleware"), "auth must reject before middlewares run")
					continue
				}
				require.Equal(t, http.StatusNoContent, resp.StatusCode)
				require.Equal(t, []string{"first", "second"}, resp.Header.Values("X-Middleware"))
				// The receiver records before it responds, so a 204 without a
				// record means something answered in its place.
				select {
				case got := <-saw:
					require.Equal(t, []string{
						"first subject=alice encoding=",
						"second subject=alice encoding=",
					}, got, "middlewares must see the authenticated client and the decoded body")
				default:
					require.FailNow(t, "the request did not reach the receiver")
				}
			}
			require.Empty(t, saw, "rejected streams must not reach the receiver")
			require.Equal(t, int64(1), srv.connections.Load(), "all streams must share one h2c connection")
		})
	}
}

// CORS wraps authentication, so allowed preflight requests are answered
// without credentials, and rejected requests still get the outer headers.
func TestConfighttpCORSPreflightRunsBeforeAuth(t *testing.T) {
	t.Parallel()

	const allowed = "https://allowed.example"
	var reached atomic.Int64
	srv := startConfighttpServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reached.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}), func(cfg *confighttp.ServerConfig) {
		withAuth(cfg)
		cfg.CORS = configoptional.Some(confighttp.CORSConfig{AllowedOrigins: []string{allowed}})
		cfg.ResponseHeaders = configopaque.MapList{{Name: "X-Served-By", Value: "collector"}}
	})

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	httpClient := priorKnowledgeClient(t)
	for _, tc := range []struct {
		name   string
		method string
		status int
	}{
		{name: "preflight", method: http.MethodOptions, status: http.StatusNoContent},
		{name: "request", method: http.MethodPost, status: http.StatusUnauthorized},
	} {
		req, err := http.NewRequestWithContext(ctx, tc.method, srv.url, http.NoBody)
		require.NoError(t, err)
		req.Header.Set("Origin", allowed)
		if tc.method == http.MethodOptions {
			req.Header.Set("Access-Control-Request-Method", http.MethodPost)
		}
		resp, err := httpClient.Do(req)
		require.NoError(t, err, tc.name)
		require.NoError(t, resp.Body.Close())
		require.Equal(t, 2, resp.ProtoMajor, tc.name)
		require.Equal(t, tc.status, resp.StatusCode, tc.name)
		require.Equal(t, allowed, resp.Header.Get("Access-Control-Allow-Origin"), tc.name)
		require.Equal(t, "collector", resp.Header.Get("X-Served-By"), tc.name)
	}
	require.Zero(t, reached.Load(), "neither request may reach the receiver")
	require.Equal(t, int64(1), srv.connections.Load(), "all streams must share one h2c connection")
}

func TestConfighttpStreamsCompleteIndependently(t *testing.T) {
	t.Parallel()

	slowStarted := make(chan struct{})
	var startOnce sync.Once
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseSlow := func() { releaseOnce.Do(func() { close(release) }) }
	srv := startConfighttpServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/slow" {
			startOnce.Do(func() { close(slowStarted) })
			select {
			case <-release:
			case <-r.Context().Done():
				return
			}
		}
		w.WriteHeader(http.StatusNoContent)
	}), nil)
	// Registered after the server, so it runs before the server is closed.
	t.Cleanup(releaseSlow)

	type result struct {
		proto  int
		status int
		err    error
	}
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	httpClient := priorKnowledgeClient(t)
	get := func(path string) result {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.url+path, http.NoBody)
		if err != nil {
			return result{err: err}
		}
		resp, err := httpClient.Do(req)
		if err != nil {
			return result{err: err}
		}
		return result{proto: resp.ProtoMajor, status: resp.StatusCode, err: resp.Body.Close()}
	}

	slowDone := make(chan result, 1)
	go func() { slowDone <- get("/slow") }()
	select {
	case <-slowStarted:
	case res := <-slowDone:
		require.FailNow(t, "the slow stream ended before reaching the handler", "%+v", res)
	case <-ctx.Done():
		require.FailNow(t, "the slow stream never reached the handler")
	}

	fast := get("/fast")
	require.NoError(t, fast.err, "the fast stream must complete while the slow stream is held")
	require.Equal(t, 2, fast.proto)
	require.Equal(t, http.StatusNoContent, fast.status)
	select {
	case res := <-slowDone:
		require.FailNow(t, "the slow stream completed before it was released", "%+v", res)
	default:
	}

	releaseSlow()
	select {
	case res := <-slowDone:
		require.NoError(t, res.err)
		require.Equal(t, 2, res.proto)
		require.Equal(t, http.StatusNoContent, res.status)
	case <-ctx.Done():
		require.FailNow(t, "the slow stream never completed")
	}
	require.Equal(t, int64(1), srv.connections.Load(), "all streams must share one h2c connection")
}

// The HTTP/1.1 Upgrade to h2c is optional (RFC 7540 Section 3.2) and is not
// supported by net/http, so the request is served as HTTP/1.1.
func TestConfighttpUpgradeServedAsHTTP1(t *testing.T) {
	t.Parallel()

	srv := startConfighttpServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), withAuth)

	for authorization, wantStatus := range map[string]string{
		"":             "401",
		"Bearer alice": "204",
	} {
		conn, err := net.Dial("tcp", srv.address)
		require.NoError(t, err)
		require.NoError(t, conn.SetDeadline(time.Now().Add(5*time.Second)))
		request := "GET / HTTP/1.1\r\nHost: " + srv.address + "\r\n" +
			"Connection: Upgrade, HTTP2-Settings\r\nUpgrade: h2c\r\nHTTP2-Settings: AAMAAABkAARAAAAAAAIAAAAA\r\n"
		if authorization != "" {
			request += "Authorization: " + authorization + "\r\n"
		}
		_, err = conn.Write([]byte(request + "\r\n"))
		require.NoError(t, err)
		statusLine, err := bufio.NewReader(conn).ReadString('\n')
		require.NoError(t, err)
		require.NoError(t, conn.Close())
		require.True(t, strings.HasPrefix(statusLine, "HTTP/1.1 "+wantStatus), statusLine)
	}
}

type testServer struct {
	server      *http.Server
	address     string
	url         string
	connections *atomic.Int64
	serveDone   <-chan struct{}
}

func withAuth(cfg *confighttp.ServerConfig) {
	cfg.Auth = configoptional.Some(confighttp.AuthConfig{
		Config: configauth.Config{AuthenticatorID: authID},
	})
}

func startConfighttpServer(t *testing.T, handler http.Handler, configure func(*confighttp.ServerConfig)) *testServer {
	t.Helper()
	return startConfighttpServerWithSettings(t, handler, configure, componenttest.NewNopTelemetrySettings())
}

func startConfighttpServerWithSettings(t *testing.T, handler http.Handler, configure func(*confighttp.ServerConfig), settings component.TelemetrySettings) *testServer {
	t.Helper()

	extensions := map[component.ID]component.Component{
		authID:    subjectAuthenticator{},
		h2cID:     &h2cExtension{},
		firstID:   recordingMiddleware{name: "first"},
		secondID:  recordingMiddleware{name: "second"},
		readingID: readingMiddleware{},
	}
	cfg := confighttp.NewDefaultServerConfig()
	cfg.NetAddr.Endpoint = "127.0.0.1:0"
	cfg.Middlewares = []configmiddleware.Config{{ID: h2cID}}
	if configure != nil {
		configure(&cfg)
	}
	listener, err := cfg.ToListener(t.Context())
	require.NoError(t, err)
	server, err := cfg.ToServer(t.Context(), extensions, settings, handler)
	require.NoError(t, err)
	connections := &atomic.Int64{}
	server.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			connections.Add(1)
		}
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = server.Serve(listener)
	}()
	t.Cleanup(func() {
		require.NoError(t, server.Close())
		<-done
	})
	address := listener.Addr().String()
	return &testServer{server: server, address: address, url: "http://" + address, connections: connections, serveDone: done}
}

func http1Client(t *testing.T) *http.Client {
	t.Helper()

	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)
	transport := &http.Transport{Protocols: protocols}
	t.Cleanup(transport.CloseIdleConnections)
	return &http.Client{Transport: transport}
}

func priorKnowledgeClient(t *testing.T) *http.Client {
	t.Helper()

	protocols := new(http.Protocols)
	protocols.SetUnencryptedHTTP2(true)
	transport := &http.Transport{Protocols: protocols}
	t.Cleanup(transport.CloseIdleConnections)
	return &http.Client{Transport: transport}
}

// recordingMiddleware marks the response with its name and records on the
// request what it saw, so tests can check where it runs in the chain.
type recordingMiddleware struct {
	component.StartFunc
	component.ShutdownFunc
	name string
}

var _ extensionmiddleware.HTTPServer = recordingMiddleware{}

func (m recordingMiddleware) GetHTTPHandler(context.Context) (extensionmiddleware.WrapHTTPHandlerFunc, error) {
	return func(_ context.Context, next http.Handler) (http.Handler, error) {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("X-Middleware", m.name)
			var subject string
			if auth := client.FromContext(r.Context()).Auth; auth != nil {
				subject, _ = auth.GetAttribute("subject").(string)
			}
			r.Header.Add("X-Middleware-Saw", m.name+" subject="+subject+" encoding="+r.Header.Get("Content-Encoding"))
			next.ServeHTTP(w, r)
		}), nil
	}, nil
}

// readingMiddleware reads the whole body before the receiver does and
// answers 413 itself when the body is over the limit.
type readingMiddleware struct {
	component.StartFunc
	component.ShutdownFunc
}

var _ extensionmiddleware.HTTPServer = readingMiddleware{}

func (readingMiddleware) GetHTTPHandler(context.Context) (extensionmiddleware.WrapHTTPHandlerFunc, error) {
	return func(_ context.Context, next http.Handler) (http.Handler, error) {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
					w.Header().Set("X-Middleware-Limited", "true")
					w.WriteHeader(http.StatusRequestEntityTooLarge)
					return
				}
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))
			next.ServeHTTP(w, r)
		}), nil
	}, nil
}

type subjectAuthenticator struct {
	component.StartFunc
	component.ShutdownFunc
}

var _ extensionauth.Server = subjectAuthenticator{}

func (subjectAuthenticator) Authenticate(ctx context.Context, sources map[string][]string) (context.Context, error) {
	for name, values := range sources {
		if !strings.EqualFold(name, "Authorization") || len(values) != 1 {
			continue
		}
		subject, ok := strings.CutPrefix(values[0], "Bearer ")
		if ok && subject != "" {
			info := client.FromContext(ctx)
			info.Auth = subjectAuthData(subject)
			return client.NewContext(ctx, info), nil
		}
	}
	return ctx, errors.New("missing bearer token")
}

type subjectAuthData string

func (subjectAuthData) GetAttributeNames() []string {
	return []string{"subject"}
}

func (s subjectAuthData) GetAttribute(name string) any {
	if name == "subject" {
		return string(s)
	}
	return nil
}
