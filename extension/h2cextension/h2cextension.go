// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package h2cextension provides an HTTP server middleware extension for serving
// cleartext HTTP/2 connections.
package h2cextension

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"

	"golang.org/x/net/http2"
	//nolint:staticcheck // The middleware API cannot set http.Server.Protocols, and that API does not handle HTTP/1.1 upgrades.
	"golang.org/x/net/http2/h2c"

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
		//nolint:staticcheck // h2c.NewHandler is required here for HTTP/1.1 Upgrade support.
		handler := h2c.NewHandler(next, &http2.Server{})
		return h2cHandler{handler: handler, connections: &e.connections}, nil
	}, nil
}

type h2cHandler struct {
	handler     http.Handler
	connections *connectionTracker
}

func (h h2cHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if isH2CConnectionRequest(r) {
		trackingWriter := &hijackTrackingResponseWriter{
			ResponseWriter: w,
			connections:    h.connections,
		}
		w = trackingWriter
		defer func() {
			if trackingWriter.conn != nil {
				_ = trackingWriter.conn.Close()
			}
		}()
	}
	h.handler.ServeHTTP(w, r)
}

func isH2CConnectionRequest(r *http.Request) bool {
	if r.Method == "PRI" {
		return true
	}
	return hasHeaderToken(r.Header, "Upgrade", "h2c") &&
		hasHeaderToken(r.Header, "Connection", "HTTP2-Settings")
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

type hijackTrackingResponseWriter struct {
	http.ResponseWriter
	connections *connectionTracker
	conn        *trackedConn
}

func (w hijackTrackingResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *hijackTrackingResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	conn, rw, err := http.NewResponseController(w.ResponseWriter).Hijack()
	if err != nil {
		return nil, nil, err
	}
	trackedConn, err := w.connections.track(conn)
	if err != nil {
		return nil, nil, err
	}
	w.conn = trackedConn
	return trackedConn, rw, nil
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
