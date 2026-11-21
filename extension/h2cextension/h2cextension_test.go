// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package h2cextension

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"
)

func TestHTTP1Passthrough(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(newH2CHandler(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})))
	t.Cleanup(server.Close)

	resp, err := http.Get(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestPriorKnowledge(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(newH2CHandler(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "ok")
	})))
	t.Cleanup(server.Close)

	protocols := new(http.Protocols)
	protocols.SetUnencryptedHTTP2(true)
	protocols.SetHTTP1(false)
	transport := &http.Transport{Protocols: protocols}
	t.Cleanup(transport.CloseIdleConnections)
	client := &http.Client{Transport: transport}

	for _, path := range []string{"/first", "/second"} {
		resp, err := client.Get(server.URL + path)
		require.NoError(t, err)
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.Equal(t, "ok", string(body))
	}
}

func TestPriorKnowledgeTruncatedPrefaceClosesHijackedConnection(t *testing.T) {
	t.Parallel()

	extension := &h2cExtension{}
	var requests atomic.Int64
	handler := newH2CHandlerWithExtension(t, extension, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		requests.Add(1)
	}))
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	conn, err := net.Dial("tcp", server.Listener.Addr().String())
	require.NoError(t, err)
	defer conn.Close()
	if deadline, ok := t.Deadline(); ok {
		require.NoError(t, conn.SetDeadline(deadline))
	}

	_, err = io.WriteString(conn, "PRI * HTTP/2.0\r\n\r\nSM\r\n")
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		extension.connections.mu.Lock()
		defer extension.connections.mu.Unlock()
		return len(extension.connections.connections) == 1
	}, 2*time.Second, time.Millisecond)

	tcpConn, ok := conn.(*net.TCPConn)
	require.True(t, ok)
	require.NoError(t, tcpConn.CloseWrite())
	require.NoError(t, conn.SetReadDeadline(time.Now().Add(2*time.Second)))

	response, err := io.ReadAll(conn)
	require.NoError(t, err)
	require.Empty(t, response)
	require.Eventually(t, func() bool {
		extension.connections.mu.Lock()
		defer extension.connections.mu.Unlock()
		return len(extension.connections.connections) == 0
	}, 2*time.Second, time.Millisecond, "hijacked connection remains tracked after handler return")
	require.Zero(t, requests.Load())
	require.NoError(t, extension.Shutdown(context.Background()))
}

func TestUpgradeSubsequentStreamsBypassOuterAuthAndBodyLimit(t *testing.T) {
	t.Parallel()

	const bodyLimit = 4
	type identityKey struct{}
	identities := make(chan string, 3)
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		identity, _ := r.Context().Value(identityKey{}).(string)
		identities <- identity
		_, _ = io.WriteString(w, strconv.Itoa(len(body)))
	})
	handler := newH2CHandler(t, inner)
	outer := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer allowed" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		r = r.WithContext(context.WithValue(r.Context(), identityKey{}, "upgrade-identity"))
		r.Body = http.MaxBytesReader(w, r.Body, bodyLimit)
		handler.ServeHTTP(w, r)
	})
	server := httptest.NewServer(outer)
	t.Cleanup(server.Close)

	conn, framer := startUpgrade(t, server.Listener.Addr().String())
	defer conn.Close()
	require.Equal(t, "upgrade-identity", <-identities)

	status, body := upgradedRequest(
		t,
		framer,
		3,
		server.Listener.Addr().String(),
		"/unauthenticated",
		nil,
		false,
	)
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, "0", body)
	require.Equal(t, "upgrade-identity", <-identities)

	status, body = upgradedRequest(
		t,
		framer,
		5,
		server.Listener.Addr().String(),
		"/oversized",
		[]byte("too large"),
		true,
	)
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, "9", body)
	require.Equal(t, "upgrade-identity", <-identities)
}

func TestShutdownClosesHijackedConnections(t *testing.T) {
	t.Parallel()

	var requests atomic.Int64
	extension := &h2cExtension{}
	handler := newH2CHandlerWithExtension(t, extension, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	conn, framer := startUpgrade(t, server.Listener.Addr().String())
	defer conn.Close()
	require.Equal(t, int64(1), requests.Load())

	require.NoError(t, extension.Shutdown(context.Background()))

	var block bytes.Buffer
	encoder := hpack.NewEncoder(&block)
	for _, field := range []hpack.HeaderField{
		{Name: ":method", Value: http.MethodGet},
		{Name: ":scheme", Value: "http"},
		{Name: ":path", Value: "/after-shutdown"},
		{Name: ":authority", Value: server.Listener.Addr().String()},
	} {
		require.NoError(t, encoder.WriteField(field))
	}
	writeErr := framer.WriteHeaders(http2.HeadersFrameParam{
		StreamID:      3,
		BlockFragment: block.Bytes(),
		EndHeaders:    true,
		EndStream:     true,
	})
	if writeErr == nil {
		require.NoError(t, conn.SetDeadline(time.Now().Add(2*time.Second)))
		for {
			frame, err := framer.ReadFrame()
			if err != nil {
				var networkErr net.Error
				require.False(t, errors.As(err, &networkErr) && networkErr.Timeout(), "h2c connection was not closed")
				break
			}
			if headers, ok := frame.(*http2.MetaHeadersFrame); ok && headers.StreamID == 3 {
				t.Fatal("received a response for a stream sent after Shutdown")
			}
		}
	}
	require.Equal(t, int64(1), requests.Load(), "later streams must not reach the handler")
}

func TestShutdownReturnsHijackedConnectionCloseErrors(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("close failed")
	extension := &h2cExtension{}
	_, err := extension.connections.track(closeErrorConn{err: wantErr})
	require.NoError(t, err)

	require.ErrorIs(t, extension.Shutdown(context.Background()), wantErr)
}

func TestShutdownReturnsHandlerCloseErrors(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("close failed")
	extension := &h2cExtension{}
	conn, err := extension.connections.track(closeErrorConn{err: wantErr})
	require.NoError(t, err)
	require.ErrorIs(t, conn.Close(), wantErr)

	require.ErrorIs(t, extension.Shutdown(context.Background()), wantErr)
}

func newH2CHandler(t *testing.T, next http.Handler) http.Handler {
	t.Helper()

	return newH2CHandlerWithExtension(t, &h2cExtension{}, next)
}

func newH2CHandlerWithExtension(t *testing.T, extension *h2cExtension, next http.Handler) http.Handler {
	t.Helper()

	middleware, err := extension.GetHTTPHandler(context.Background())
	require.NoError(t, err)
	handler, err := middleware(context.Background(), next)
	require.NoError(t, err)
	return handler
}

type closeErrorConn struct {
	net.Conn
	err error
}

func (c closeErrorConn) Close() error {
	return c.err
}

func startUpgrade(t *testing.T, address string) (net.Conn, *http2.Framer) {
	t.Helper()

	conn, err := net.Dial("tcp", address)
	require.NoError(t, err)
	if deadline, ok := t.Deadline(); ok {
		require.NoError(t, conn.SetDeadline(deadline))
	}

	_, err = fmt.Fprintf(
		conn,
		"GET /upgrade HTTP/1.1\r\nHost: %s\r\nConnection: Upgrade, HTTP2-Settings\r\n"+
			"Upgrade: h2c\r\nHTTP2-Settings: AAMAAABk\r\nAuthorization: Bearer allowed\r\n\r\n",
		address,
	)
	require.NoError(t, err)

	reader := bufio.NewReader(conn)
	statusLine, err := reader.ReadString('\n')
	require.NoError(t, err)
	require.Contains(t, statusLine, "101")
	for {
		line, readErr := reader.ReadString('\n')
		require.NoError(t, readErr)
		if line == "\r\n" {
			break
		}
	}

	_, err = io.WriteString(conn, http2.ClientPreface)
	require.NoError(t, err)
	framer := http2.NewFramer(conn, bufferedConn{Conn: conn, reader: reader})
	framer.ReadMetaHeaders = hpack.NewDecoder(4096, nil)
	require.NoError(t, framer.WriteSettings())

	for {
		frame, readErr := framer.ReadFrame()
		require.NoError(t, readErr)
		headers, ok := frame.(*http2.MetaHeadersFrame)
		if !ok || headers.StreamID != 1 {
			continue
		}
		if headers.StreamEnded() {
			break
		}
		for {
			frame, readErr = framer.ReadFrame()
			require.NoError(t, readErr)
			data, ok := frame.(*http2.DataFrame)
			if ok && data.StreamID == 1 && data.StreamEnded() {
				break
			}
		}
		break
	}
	return conn, framer
}

func upgradedRequest(
	t *testing.T,
	framer *http2.Framer,
	streamID uint32,
	authority,
	path string,
	body []byte,
	authorized bool,
) (int, string) {
	t.Helper()

	var block bytes.Buffer
	encoder := hpack.NewEncoder(&block)
	fields := []hpack.HeaderField{
		{Name: ":method", Value: http.MethodGet},
		{Name: ":scheme", Value: "http"},
		{Name: ":path", Value: path},
		{Name: ":authority", Value: authority},
	}
	if len(body) > 0 {
		fields[0].Value = http.MethodPost
		fields = append(fields, hpack.HeaderField{Name: "content-length", Value: strconv.Itoa(len(body))})
	}
	if authorized {
		fields = append(fields, hpack.HeaderField{Name: "authorization", Value: "Bearer allowed"})
	}
	for _, field := range fields {
		require.NoError(t, encoder.WriteField(field))
	}

	require.NoError(t, framer.WriteHeaders(http2.HeadersFrameParam{
		StreamID:      streamID,
		BlockFragment: block.Bytes(),
		EndHeaders:    true,
		EndStream:     len(body) == 0,
	}))
	if len(body) > 0 {
		require.NoError(t, framer.WriteData(streamID, true, body))
	}

	for {
		frame, readErr := framer.ReadFrame()
		require.NoError(t, readErr)
		headers, ok := frame.(*http2.MetaHeadersFrame)
		if !ok || headers.StreamID != streamID {
			continue
		}

		status := 0
		for _, field := range headers.Fields {
			if field.Name == ":status" {
				_, err := fmt.Sscanf(field.Value, "%d", &status)
				require.NoError(t, err)
			}
		}
		if headers.StreamEnded() {
			return status, ""
		}
		for {
			frame, readErr = framer.ReadFrame()
			require.NoError(t, readErr)
			data, ok := frame.(*http2.DataFrame)
			if ok && data.StreamID == streamID {
				return status, strings.TrimSpace(string(data.Data()))
			}
		}
	}
}

type bufferedConn struct {
	net.Conn
	reader *bufio.Reader
}

func (c bufferedConn) Read(p []byte) (int, error) {
	return c.reader.Read(p)
}
