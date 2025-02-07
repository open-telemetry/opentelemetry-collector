// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/multierr"
)

func TestMergedContextDeadline(t *testing.T) {
	now := time.Now()
	ctx1 := context.Background()
	mergedContext := NewMergedContext(ctx1)

	deadline, ok := mergedContext.Deadline()
	require.False(t, ok)

	ctx2, cancel2 := context.WithDeadline(context.Background(), now.Add(200))
	defer cancel2()
	mergedContext.Merge(ctx2)

	deadline, ok = mergedContext.Deadline()
	require.True(t, ok)
	require.Equal(t, now.Add(200), deadline)

	ctx3, cancel3 := context.WithDeadline(context.Background(), now.Add(300))
	defer cancel3()
	ctx4, cancel4 := context.WithDeadline(context.Background(), now.Add(100))
	defer cancel4()
	mergedContext.Merge(ctx3)
	mergedContext.Merge(ctx4)

	deadline, ok = mergedContext.Deadline()
	require.True(t, ok)
	require.Equal(t, now.Add(100), deadline)
}

func TestMergedContextDone(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(100))
	mergedContext := NewMergedContext(ctx)

	defer cancel()
	neverReady := make(chan struct{})
	select {
	case <-neverReady:
	case <-mergedContext.Done():
	}
}

func TestMergedContextDone2(t *testing.T) {
	ctx1 := context.Background()
	ctx2, _ := context.WithDeadline(context.Background(), time.Now().Add(100))
	mergedCtx := NewMergedContext(ctx1)
	mergedCtx.Merge(ctx2)

	neverReady := make(chan struct{})
	select {
	case <-neverReady:
	case <-mergedCtx.Done():
	}
}

func TestMergedContextDone3(t *testing.T) {
	ctx1, _ := context.WithCancel(context.Background())
	ctx2, _ := context.WithDeadline(context.Background(), time.Now().Add(100))
	mergedCtx := NewMergedContext(ctx1)
	mergedCtx.Merge(ctx2)

	neverReady := make(chan struct{})
	select {
	case <-neverReady:
	case <-mergedCtx.Done():
	}
}

func TestMergedContextErr(t *testing.T) {
	ctx1, cancel1 := context.WithCancelCause(context.Background())
	ctx2, cancel2 := context.WithCancelCause(context.Background())
	mergedCtx := NewMergedContext(ctx1)
	mergedCtx.Merge(ctx2)

	cancel1(fmt.Errorf("cancel cause 1"))
	cancel2(fmt.Errorf("cancel cause 2"))

	var err error
	err = multierr.Append(err, ctx1.Err())
	err = multierr.Append(err, ctx2.Err())
	require.Equal(t, err, mergedCtx.Err())
}

func TestMergedContextValue(t *testing.T) {
	ctx1 := context.WithValue(
		context.Background(),
		"key1",
		"value1")
	ctx2 := context.WithValue(
		context.Background(),
		"key2",
		"value2")
	ctx3 := context.WithValue(
		context.Background(),
		"key2",
		"value3")

	mergedCtx := NewMergedContext(ctx1)
	mergedCtx.Merge(ctx2)
	mergedCtx.Merge(ctx3)
	require.Equal(t, "value1", mergedCtx.Value("key1"))
	require.Equal(t, "value2", mergedCtx.Value("key2"))
	require.Equal(t, nil, mergedCtx.Value("key3"))
	require.Equal(t, mergedCtx.Err(), multierr.Append(ctx1.Err(), ctx2.Err()))
}
