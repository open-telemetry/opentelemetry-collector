// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlpreceiver

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/internal/testutil"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
	"go.opentelemetry.io/collector/pdata/testdata"
	"go.opentelemetry.io/collector/receiver/otlpreceiver/internal/metadata"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

const (
	itemsPerRequest     = 10_000
	protobufContentType = "application/x-protobuf"
)

func startLogsReceiver(b *testing.B, cfg *Config, sink *consumertest.LogsSink) {
	set := receivertest.NewNopSettings(metadata.Type)
	factory := NewFactory()
	r, err := factory.CreateLogs(b.Context(), set, cfg, sink)
	require.NoError(b, err)

	require.NoError(b, r.Start(b.Context(), componenttest.NewNopHost()))
	b.Cleanup(func() {
		require.NoError(b, r.Shutdown(context.Background()))
	})
}

// BenchmarkGRPCLogsSequential benchmarks sequentially receiving logs over OTLP/gRPC.
// A typical deployment would receive multiple concurrent requests, this benchmark tries to
// measure the performance of the receiver without concurrency to have a more stable benchmark.
func BenchmarkGRPCLogsSequential(b *testing.B) {
	endpoint := testutil.GetAvailableLocalAddress(b)
	cfg := createDefaultConfig().(*Config)
	cfg.GRPC.GetOrInsertDefault().NetAddr.Endpoint = endpoint
	var sink consumertest.LogsSink
	startLogsReceiver(b, cfg, &sink)

	cc, err := grpc.NewClient(endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(b, err)
	b.Cleanup(func() { require.NoError(b, cc.Close()) })
	logClient := plogotlp.NewGRPCClient(cc)
	req := plogotlp.NewExportRequestFromLogs(testdata.GenerateLogs(itemsPerRequest))

	for b.Loop() {
		_, err := logClient.Export(b.Context(), req)
		require.NoError(b, err)
	}

	require.Equal(b, b.N*itemsPerRequest, sink.LogRecordCount())
}

// BenchmarkHTTPProtoLogsSequential benchmarks sequentially receiving logs over OTLP/HTTP (proto).
// A typical deployment would receive multiple concurrent requests, this benchmark tries to
// measure the performance of the receiver without concurrency to have a more stable benchmark.
func BenchmarkHTTPProtoLogsSequential(b *testing.B) {
	endpoint := testutil.GetAvailableLocalAddress(b)
	cfg := createDefaultConfig().(*Config)
	cfg.HTTP.GetOrInsertDefault().ServerConfig.NetAddr.Endpoint = endpoint
	var sink consumertest.LogsSink
	startLogsReceiver(b, cfg, &sink)

	marshaler := &plog.ProtoMarshaler{}
	bodyBytes, err := marshaler.MarshalLogs(testdata.GenerateLogs(itemsPerRequest))
	require.NoError(b, err)
	req, err := http.NewRequest(http.MethodPost, "http://"+endpoint+defaultLogsURLPath, bytes.NewReader(bodyBytes))
	require.NoError(b, err)
	req.Header.Set("Content-Type", protobufContentType)

	reader := bytes.NewReader(bodyBytes)
	for b.Loop() {
		reader.Reset(bodyBytes)
		req.Body = io.NopCloser(reader)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(b, err)
		require.Equal(b, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	}

	require.Equal(b, b.N*itemsPerRequest, sink.LogRecordCount())
}
