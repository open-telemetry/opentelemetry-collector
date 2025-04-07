// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch

import (
	"context"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/client"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requesttest"
)

func TestPartitioner_GetKeyFromRequest(t *testing.T) {
	partitioner := NewPartitioner(func(_ context.Context, req request.Request) string {
		return strconv.Itoa(req.(*requesttest.FakeRequest).ItemsCount())
	})

	require.Equal(t, "2", partitioner.GetKey(context.Background(), &requesttest.FakeRequest{Items: 2}))
	require.Equal(t, "3", partitioner.GetKey(context.Background(), &requesttest.FakeRequest{Items: 3}))
	require.Equal(t, "4", partitioner.GetKey(context.Background(), &requesttest.FakeRequest{Items: 4}))
}

func TestPartitioner_GetKeyFromContext(t *testing.T) {
	partitioner := NewPartitioner(func(ctx context.Context, _ request.Request) string {
		return client.FromContext(ctx).Metadata.Get("metadata_key")[0]
	})

	ctx1 := client.NewContext(context.Background(), client.Info{
		Metadata: client.NewMetadata(map[string][]string{"metadata_key": {"partition1"}}),
	})
	require.Equal(t, "partition1", partitioner.GetKey(ctx1, &requesttest.FakeRequest{Items: 2}))

	ctx2 := client.NewContext(context.Background(), client.Info{
		Metadata: client.NewMetadata(map[string][]string{"metadata_key": {"partition2"}}),
	})
	require.Equal(t, "partition2", partitioner.GetKey(ctx2, &requesttest.FakeRequest{Items: 2}))
}
