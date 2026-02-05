// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlpexporter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/internal/testutil"
)

func TestConnPool_Creation(t *testing.T) {
	addr := testutil.GetAvailableLocalAddress(t)

	clientCfg := configgrpc.NewDefaultClientConfig()
	clientCfg.Endpoint = addr

	ctx := context.Background()
	host := componenttest.NewNopHost()
	settings := componenttest.NewNopTelemetrySettings()

	t.Run("single connection", func(t *testing.T) {
		pool, err := newConnPool(ctx, 1, clientCfg, host, settings)
		require.NoError(t, err)
		require.NotNil(t, pool)
		assert.Len(t, pool.conns, 1)
		assert.NotNil(t, pool.conns[0])

		conn := pool.getConn()
		assert.NotNil(t, conn)
		assert.Equal(t, pool.conns[0], conn)

		require.NoError(t, pool.Close())
	})

	t.Run("multiple connections", func(t *testing.T) {
		pool, err := newConnPool(ctx, 4, clientCfg, host, settings)
		require.NoError(t, err)
		require.NotNil(t, pool)
		assert.Len(t, pool.conns, 4)

		for i, conn := range pool.conns {
			assert.NotNil(t, conn, "connection %d should not be nil", i)
		}

		require.NoError(t, pool.Close())
	})

	t.Run("zero size defaults to one", func(t *testing.T) {
		pool, err := newConnPool(ctx, 0, clientCfg, host, settings)
		require.NoError(t, err)
		require.NotNil(t, pool)
		assert.Len(t, pool.conns, 1)

		require.NoError(t, pool.Close())
	})

	t.Run("negative size defaults to one", func(t *testing.T) {
		pool, err := newConnPool(ctx, -5, clientCfg, host, settings)
		require.NoError(t, err)
		require.NotNil(t, pool)
		assert.Len(t, pool.conns, 1)

		require.NoError(t, pool.Close())
	})
}

func TestConnPool_RoundRobin(t *testing.T) {
	addr := testutil.GetAvailableLocalAddress(t)

	clientCfg := configgrpc.NewDefaultClientConfig()
	clientCfg.Endpoint = addr

	ctx := context.Background()
	host := componenttest.NewNopHost()
	settings := componenttest.NewNopTelemetrySettings()

	pool, err := newConnPool(ctx, 4, clientCfg, host, settings)
	require.NoError(t, err)
	require.NotNil(t, pool)
	defer pool.Close()

	// Verify round-robin distribution
	// Since we start at 0 and increment, first call returns conns[1], then conns[2], etc.
	seen := make(map[*grpc.ClientConn]int)
	iterations := 100

	for i := 0; i < iterations; i++ {
		conn := pool.getConn()
		seen[conn]++
	}

	// Each connection should be used approximately equally
	expectedPerConn := iterations / len(pool.conns)
	for conn, count := range seen {
		assert.NotNil(t, conn)
		// Allow some variance but should be relatively balanced
		assert.Greater(t, count, 0, "connection should be used at least once")
		assert.InDelta(t, expectedPerConn, count, float64(iterations)*0.3, "distribution should be relatively balanced")
	}

	// All connections should have been used
	assert.Len(t, seen, len(pool.conns), "all connections should be used")
}

func TestConnPool_Close(t *testing.T) {
	addr := testutil.GetAvailableLocalAddress(t)

	clientCfg := configgrpc.NewDefaultClientConfig()
	clientCfg.Endpoint = addr

	ctx := context.Background()
	host := componenttest.NewNopHost()
	settings := componenttest.NewNopTelemetrySettings()

	pool, err := newConnPool(ctx, 3, clientCfg, host, settings)
	require.NoError(t, err)
	require.NotNil(t, pool)

	// Close should succeed
	err = pool.Close()
	assert.NoError(t, err)

	// All connections should be nil after close
	for i, conn := range pool.conns {
		assert.Nil(t, conn, "connection %d should be nil after close", i)
	}

	// Second close should also succeed (idempotent)
	err = pool.Close()
	assert.NoError(t, err)
}

func TestConnPool_SingleConnectionFastPath(t *testing.T) {
	addr := testutil.GetAvailableLocalAddress(t)

	clientCfg := configgrpc.NewDefaultClientConfig()
	clientCfg.Endpoint = addr

	ctx := context.Background()
	host := componenttest.NewNopHost()
	settings := componenttest.NewNopTelemetrySettings()

	pool, err := newConnPool(ctx, 1, clientCfg, host, settings)
	require.NoError(t, err)
	require.NotNil(t, pool)
	defer pool.Close()

	// With a single connection, getConn should always return the same connection
	conn1 := pool.getConn()
	conn2 := pool.getConn()
	conn3 := pool.getConn()

	assert.Equal(t, conn1, conn2, "single connection should always return same connection")
	assert.Equal(t, conn2, conn3, "single connection should always return same connection")
}
