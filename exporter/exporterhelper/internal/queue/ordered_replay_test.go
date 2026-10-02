// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/experr"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/hosttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/storagetest"
)

func TestPersistentQueueReplayOrderIsOptIn(t *testing.T) {
	for _, ordered := range []bool{false, true} {
		name := "existing"
		if ordered {
			name = "ordered"
		}
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			host := hosttest.NewHost(map[component.ID]component.Component{{}: storagetest.NewMockStorageExtension(nil)})
			set := newSettingsWithStorage(request.SizerTypeRequests, 10)
			set.ReplayInOrder = ordered
			start := func() *persistentQueue[intRequest] {
				t.Helper()
				q := newPersistentQueue[intRequest](set).(*persistentQueue[intRequest])
				require.NoError(t, q.Start(ctx, host))
				return q
			}
			q := start()
			for _, value := range []intRequest{1, 2, 3} {
				require.NoError(t, q.Offer(ctx, value))
			}
			for range 2 {
				_, _, done, ok := q.Read(ctx)
				require.True(t, ok)
				done.OnDone(experr.NewShutdownErr(context.Canceled))
			}
			require.NoError(t, q.Shutdown(ctx))

			restarted := start()
			require.EqualValues(t, 3, restarted.Size())
			_, value, done, ok := restarted.Read(ctx)
			require.True(t, ok)
			if ordered {
				require.Equal(t, intRequest(1), value, "older in-flight requests must precede queued work")
				require.EqualValues(t, 0, done.(*indexDone).index)
			} else {
				require.Equal(t, intRequest(3), value, "existing recovery appends in-flight requests to the tail")
				require.EqualValues(t, 2, done.(*indexDone).index)
				require.Empty(t, restarted.replayItems)
			}
			done.OnDone(experr.NewShutdownErr(context.Canceled))
			require.NoError(t, restarted.Shutdown(ctx))

			// Restart again with a replay still in flight and, in ordered mode,
			// another original item remaining in the persisted replay list.
			restarted = start()
			for i, want := range []intRequest{1, 2, 3} {
				_, value, done, ok = restarted.Read(ctx)
				require.True(t, ok)
				require.Equal(t, want, value)
				index := i + 3
				if ordered {
					index = i
				}
				require.EqualValues(t, index, done.(*indexDone).index)
				done.OnDone(nil)
			}
			require.Zero(t, restarted.Size())
			require.NoError(t, restarted.Shutdown(ctx))
		})
	}
}
