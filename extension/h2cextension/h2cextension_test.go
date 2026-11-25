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
)

var (
	h2cID  = component.MustNewID("h2c")
	authID = component.MustNewID("testauth")
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
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.url, http.NoBody)
	require.NoError(t, err)
	resp, err := httpClient.Do(req)
	if err == nil {
		require.NoError(t, resp.Body.Close())
	}
	require.Error(t, err, "new streams must be refused after shutdown starts")
	require.Equal(t, int64(1), entries.Load(), "a new stream reached the handler after shutdown started")

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
		authID: subjectAuthenticator{},
		h2cID:  &h2cExtension{},
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
