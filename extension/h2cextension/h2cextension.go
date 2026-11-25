// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package h2cextension provides an HTTP server middleware extension for serving
// cleartext HTTP/2 connections.
package h2cextension

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"

	"golang.org/x/net/http2"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/extensionmiddleware"
)

var (
	_ extensionmiddleware.HTTPServer = (*h2cExtension)(nil)
	_ component.Component            = (*h2cExtension)(nil)
)

var errExtensionShutdown = errors.New("h2c extension is shut down")

type h2cExtension struct {
	connections connectionTracker
}

func (*h2cExtension) Start(context.Context, component.Host) error {
	return nil
}

func (e *h2cExtension) Shutdown(context.Context) error {
	return e.connections.closeAll()
}

// GetHTTPHandler implements extensionmiddleware.HTTPServer.
func (e *h2cExtension) GetHTTPHandler(context.Context) (extensionmiddleware.WrapHTTPHandlerFunc, error) {
	return func(_ context.Context, next http.Handler) (http.Handler, error) {
		return &h2cHandler{next: next, h2: &http2.Server{}, connections: &e.connections}, nil
	}, nil
}

// h2cConnectionKey marks requests that arrive as streams on an h2c connection.
type h2cConnectionKey struct{}

// upgradeStreamKey marks the HTTP/1.1 Upgrade request that HTTP/2 serves as stream 1.
type upgradeStreamKey struct{}

type h2cHandler struct {
	next        http.Handler
	h2          *http2.Server
	connections *connectionTracker
}

func (h *h2cHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Context().Value(h2cConnectionKey{}) != nil {
		h.next.ServeHTTP(w, r)
		return
	}
	switch {
	case isPriorKnowledgeRequest(r):
		h.servePriorKnowledge(w, r)
	case isUpgradeRequest(r):
		h.serveUpgrade(w, r)
	default:
		h.next.ServeHTTP(w, r)
	}
}

// servePriorKnowledge handles h2c with prior knowledge (RFC 9113 Section 3.3).
func (h *h2cHandler) servePriorKnowledge(w http.ResponseWriter, r *http.Request) {
	server := serverFromRequest(r)
	if server == nil {
		http.Error(w, "h2c requires an HTTP server handler", http.StatusHTTPVersionNotSupported)
		return
	}
	conn, rw, err := h.hijack(w)
	if err != nil {
		return
	}
	defer conn.Close()

	preface := make([]byte, len(priorKnowledgePrefaceBody))
	if _, err := io.ReadFull(rw, preface); err != nil || string(preface) != priorKnowledgePrefaceBody {
		return
	}
	h.serveConn(r.Context(), server, newBufferedConn(conn, rw), nil, nil)
}

// serveUpgrade handles the HTTP/1.1 Upgrade to h2c (RFC 7540 Section 3.2).
func (h *h2cHandler) serveUpgrade(w http.ResponseWriter, r *http.Request) {
	server := serverFromRequest(r)
	if server == nil {
		// Without the server handler, later streams could not be routed through
		// the server's full handler chain, so ignore the optional upgrade.
		h.next.ServeHTTP(w, r)
		return
	}
	settings, err := h2cSettings(r.Header)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// The upgrade request is served as stream 1 after the connection switches
	// protocols, so its body must be read before the connection is hijacked.
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	r.Body = io.NopCloser(bytes.NewReader(body))

	conn, rw, err := h.hijack(w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	if _, err := rw.WriteString("HTTP/1.1 101 Switching Protocols\r\nConnection: Upgrade\r\nUpgrade: h2c\r\n\r\n"); err != nil {
		return
	}
	h.serveConn(r.Context(), server, newBufferedConn(conn, rw), r, settings)
}

func (h *h2cHandler) hijack(w http.ResponseWriter) (net.Conn, *bufio.ReadWriter, error) {
	conn, rw, err := http.NewResponseController(w).Hijack()
	if err != nil {
		return nil, nil, err
	}
	tracked, err := h.connections.track(conn)
	if err != nil {
		return nil, nil, err
	}
	return tracked, rw, nil
}

// serveConn serves HTTP/2 on a hijacked connection. Every stream, except the
// Upgrade request served as stream 1, re-enters the server's handler so it
// passes through the same handler chain as an HTTP/1.1 request. Stream contexts
// do not inherit values, such as authenticated identity, from the request that
// started the connection.
func (h *h2cHandler) serveConn(parent context.Context, server *http.Server, conn net.Conn, upgrade *http.Request, settings []byte) {
	ctx, cancel := context.WithCancel(context.WithValue(context.Background(), h2cConnectionKey{}, struct{}{}))
	defer cancel()
	stop := context.AfterFunc(parent, cancel)
	defer stop()

	dispatcher := &streamDispatcher{server: server, upgradeHandler: h.next}
	if upgrade != nil {
		upgrade = upgrade.WithContext(context.WithValue(upgrade.Context(), upgradeStreamKey{}, dispatcher))
	}
	//nolint:staticcheck // http.Server.Protocols cannot be set from a handler middleware and does not support HTTP/1.1 upgrades.
	h.h2.ServeConn(conn, &http2.ServeConnOpts{
		Context:          ctx,
		BaseConfig:       server,
		Handler:          dispatcher,
		UpgradeRequest:   upgrade,
		Settings:         settings,
		SawClientPreface: upgrade == nil,
	})
}

type streamDispatcher struct {
	server         *http.Server
	upgradeHandler http.Handler
}

func (d *streamDispatcher) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// The Upgrade request already passed through the server's handler chain
	// before it reached this middleware.
	if r.Context().Value(upgradeStreamKey{}) == d {
		d.upgradeHandler.ServeHTTP(w, r)
		return
	}
	d.server.Handler.ServeHTTP(w, r)
}

func serverFromRequest(r *http.Request) *http.Server {
	server, ok := r.Context().Value(http.ServerContextKey).(*http.Server)
	if !ok || server == nil || server.Handler == nil {
		return nil
	}
	return server
}

const priorKnowledgePrefaceBody = "SM\r\n\r\n"

func isPriorKnowledgeRequest(r *http.Request) bool {
	return r.Method == "PRI" && len(r.Header) == 0 && r.URL.Path == "*" && r.Proto == "HTTP/2.0"
}

func isUpgradeRequest(r *http.Request) bool {
	return hasHeaderToken(r.Header, "Upgrade", "h2c") &&
		hasHeaderToken(r.Header, "Connection", "HTTP2-Settings")
}

func h2cSettings(header http.Header) ([]byte, error) {
	values := header.Values("HTTP2-Settings")
	if len(values) != 1 {
		return nil, fmt.Errorf("expected one HTTP2-Settings header, got %d", len(values))
	}
	return base64.RawURLEncoding.DecodeString(values[0])
}

func hasHeaderToken(header http.Header, name, wanted string) bool {
	for _, value := range header.Values(name) {
		for token := range strings.SplitSeq(value, ",") {
			if strings.EqualFold(strings.TrimSpace(token), wanted) {
				return true
			}
		}
	}
	return false
}

// newBufferedConn returns conn, preserving any bytes already buffered by the
// HTTP/1.1 server.
func newBufferedConn(conn net.Conn, rw *bufio.ReadWriter) net.Conn {
	_ = rw.Flush()
	if rw.Reader.Buffered() == 0 {
		return conn
	}
	return &bufferedConn{Conn: conn, reader: rw.Reader}
}

type bufferedConn struct {
	net.Conn
	reader *bufio.Reader
}

func (c *bufferedConn) Read(p []byte) (int, error) {
	if c.reader != nil {
		if c.reader.Buffered() > 0 {
			return c.reader.Read(p[:min(len(p), c.reader.Buffered())])
		}
		c.reader = nil
	}
	return c.Conn.Read(p)
}

type connectionTracker struct {
	mu          sync.Mutex
	connections map[*trackedConn]struct{}
	closeErrors []error
	shutdown    bool
}

func (t *connectionTracker) track(conn net.Conn) (*trackedConn, error) {
	tracked := &trackedConn{Conn: conn, tracker: t}

	t.mu.Lock()
	if t.shutdown {
		t.mu.Unlock()
		return nil, errors.Join(errExtensionShutdown, conn.Close())
	}
	if t.connections == nil {
		t.connections = make(map[*trackedConn]struct{})
	}
	t.connections[tracked] = struct{}{}
	t.mu.Unlock()
	return tracked, nil
}

func (t *connectionTracker) remove(conn *trackedConn, closeErr error) {
	t.mu.Lock()
	delete(t.connections, conn)
	if closeErr != nil && !t.shutdown {
		t.closeErrors = append(t.closeErrors, fmt.Errorf("closing hijacked h2c connection: %w", closeErr))
	}
	t.mu.Unlock()
}

func (t *connectionTracker) closeAll() error {
	t.mu.Lock()
	t.shutdown = true
	errs := t.closeErrors
	t.closeErrors = nil
	connections := make([]*trackedConn, 0, len(t.connections))
	for conn := range t.connections {
		connections = append(connections, conn)
	}
	t.mu.Unlock()

	for _, conn := range connections {
		if err := conn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("closing hijacked h2c connection: %w", err))
		}
	}
	return errors.Join(errs...)
}

type trackedConn struct {
	net.Conn
	tracker *connectionTracker
	once    sync.Once
	err     error
}

func (c *trackedConn) Close() error {
	c.once.Do(func() {
		c.err = c.Conn.Close()
		c.tracker.remove(c, c.err)
	})
	return c.err
}
