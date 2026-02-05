// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlpexporter // import "go.opentelemetry.io/collector/exporter/otlpexporter"

import (
	"context"
	"sync"
	"sync/atomic"

	"google.golang.org/grpc"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configgrpc"
)

// connPool manages a pool of gRPC client connections for improved throughput
// in high-latency or high-throughput scenarios.
type connPool struct {
	// conns holds all the gRPC connections in the pool
	conns []*grpc.ClientConn

	// counter is used for round-robin selection across connections
	counter atomic.Uint64

	// mu protects shutdown operations
	mu sync.Mutex
}

// newConnPool creates a new connection pool with the specified size.
// All connections are created immediately using the provided configuration.
func newConnPool(
	ctx context.Context,
	size int,
	clientConfig configgrpc.ClientConfig,
	host component.Host,
	settings component.TelemetrySettings,
	opts ...configgrpc.ToClientConnOption,
) (*connPool, error) {
	if size <= 0 {
		size = 1
	}

	pool := &connPool{
		conns: make([]*grpc.ClientConn, size),
	}

	// Create all connections upfront
	for i := 0; i < size; i++ {
		conn, err := clientConfig.ToClientConn(ctx, host.GetExtensions(), settings, opts...)
		if err != nil {
			// On error, close any successfully created connections
			pool.Close()
			return nil, err
		}
		pool.conns[i] = conn
	}

	return pool, nil
}

// getConn returns a connection from the pool using round-robin selection.
// This method is thread-safe and lock-free.
func (p *connPool) getConn() *grpc.ClientConn {
	if len(p.conns) == 1 {
		// Fast path for single connection (most common case)
		return p.conns[0]
	}

	// Round-robin selection using atomic counter
	idx := p.counter.Add(1) % uint64(len(p.conns))
	return p.conns[idx]
}

// Close closes all connections in the pool.
// This method is safe to call multiple times.
func (p *connPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var firstErr error
	for i, conn := range p.conns {
		if conn != nil {
			if err := conn.Close(); err != nil && firstErr == nil {
				firstErr = err
			}
			p.conns[i] = nil
		}
	}
	return firstErr
}
