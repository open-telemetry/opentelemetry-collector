// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package memorylimiterextension

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	embeddedmetric "go.opentelemetry.io/otel/metric/embedded"
	noopmetric "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/extension/extensiontest"
	"go.opentelemetry.io/collector/extension/memorylimiterextension/internal/metadata"
	"go.opentelemetry.io/collector/extension/memorylimiterextension/internal/metadatatest"
	"go.opentelemetry.io/collector/internal/memorylimiter"
	"go.opentelemetry.io/collector/internal/memorylimiter/iruntime"
)

type errorMeter struct {
	noopmetric.Meter
}

type errorMeterProvider struct {
	embeddedmetric.MeterProvider
}

func (errorMeterProvider) Meter(string, ...metric.MeterOption) metric.Meter {
	return errorMeter{}
}

func (errorMeter) Int64Counter(string, ...metric.Int64CounterOption) (metric.Int64Counter, error) {
	return nil, errors.New("failed to create counter")
}

type mockServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (m *mockServerStream) Context() context.Context {
	return m.ctx
}

func TestCreateExtension(t *testing.T) {
	factory := NewFactory()
	cfg := &Config{
		CheckInterval:         time.Second,
		MemoryLimitPercentage: 99,
		MemorySpikePercentage: 99,
	}

	set := extensiontest.NewNopSettings(extensiontest.NopType)
	set.ID = component.NewID(component.MustNewType("memory_limiter"))

	ext, err := factory.Create(context.Background(), set, cfg)

	require.NoError(t, err)
	require.NotNil(t, ext)
}

func newRefusingMemoryLimiter(t *testing.T, tb *metadata.TelemetryBuilder) *memoryLimiterExtension {
	t.Helper()

	memorylimiter.GetMemoryFn = func() (uint64, error) {
		return 2048, nil
	}
	memorylimiter.ReadMemStatsFn = func(ms *runtime.MemStats) {
		ms.Alloc = 1800
	}

	t.Cleanup(func() {
		memorylimiter.GetMemoryFn = iruntime.TotalMemory
		memorylimiter.ReadMemStatsFn = runtime.ReadMemStats
	})

	cfg := &Config{
		CheckInterval:         time.Second,
		MemoryLimitPercentage: 50,
		MemorySpikePercentage: 1,
	}

	ml, err := newMemoryLimiter(cfg, zap.NewNop(), tb)
	require.NoError(t, err)

	ml.memLimiter.CheckMemLimits()
	require.True(t, ml.MustRefuse())

	return ml
}

func TestNewMemoryLimiter_Error(t *testing.T) {
	originalGetMemoryFn := memorylimiter.GetMemoryFn
	t.Cleanup(func() {
		memorylimiter.GetMemoryFn = originalGetMemoryFn
	})

	memorylimiter.GetMemoryFn = func() (uint64, error) {
		return 0, errors.New("failed to get total memory")
	}

	cfg := &Config{
		CheckInterval:         time.Second,
		MemoryLimitPercentage: 80,
		MemorySpikePercentage: 10,
	}

	telemetryBuilder, err := metadata.NewTelemetryBuilder(
		componenttest.NewNopTelemetrySettings(),
	)
	require.NoError(t, err)

	ext, err := newMemoryLimiter(cfg, zap.NewNop(), telemetryBuilder)

	require.Error(t, err)
	require.Nil(t, ext)
	require.ErrorContains(t, err, "failed to get total memory")
}

func TestMemoryPressureResponse(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		mlCfg       *Config
		memAlloc    uint64
		expectError bool
	}{
		{
			name: "Below memAllocLimit",
			mlCfg: &Config{
				CheckInterval:         time.Second,
				MemoryLimitPercentage: 50,
				MemorySpikePercentage: 1,
			},
			memAlloc:    800,
			expectError: false,
		},
		{
			name: "Above memAllocLimit",
			mlCfg: &Config{
				CheckInterval:         time.Second,
				MemoryLimitPercentage: 50,
				MemorySpikePercentage: 1,
			},
			memAlloc:    1800,
			expectError: true,
		},
		{
			name: "Below memSpikeLimit",
			mlCfg: &Config{
				CheckInterval:         time.Second,
				MemoryLimitPercentage: 50,
				MemorySpikePercentage: 10,
			},
			memAlloc:    800,
			expectError: false,
		},
		{
			name: "Above memSpikeLimit",
			mlCfg: &Config{
				CheckInterval:         time.Second,
				MemoryLimitPercentage: 50,
				MemorySpikePercentage: 11,
			},
			memAlloc:    800,
			expectError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			memorylimiter.GetMemoryFn = func() (uint64, error) {
				return uint64(2048), nil
			}
			memorylimiter.ReadMemStatsFn = func(ms *runtime.MemStats) {
				ms.Alloc = tt.memAlloc
			}
			t.Cleanup(func() {
				memorylimiter.GetMemoryFn = iruntime.TotalMemory
				memorylimiter.ReadMemStatsFn = runtime.ReadMemStats
			})

			tb, err := metadata.NewTelemetryBuilder(componenttest.NewNopTelemetrySettings())
			require.NoError(t, err)

			ml, err := newMemoryLimiter(tt.mlCfg, zap.NewNop(), tb)
			assert.NoError(t, err)

			assert.NoError(t, ml.Start(ctx, componenttest.NewNopHost()))
			ml.memLimiter.CheckMemLimits()
			mustRefuse := ml.MustRefuse()
			if tt.expectError {
				assert.True(t, mustRefuse)
			} else {
				require.NoError(t, err)
			}
			assert.NoError(t, ml.Shutdown(ctx))
		})
	}
}

func TestCreateExtension_TelemetryBuilderError(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	settings := extensiontest.NewNopSettings(extensiontest.NopType)
	settings.ID = component.NewID(component.MustNewType("memory_limiter"))
	settings.MeterProvider = errorMeterProvider{}

	ext, err := factory.Create(context.Background(), settings, cfg)

	require.Error(t, err)
	require.Nil(t, ext)
}

func TestGRPCUnaryInterceptor_Normal(t *testing.T) {
	ctx := context.Background()

	tb, err := metadata.NewTelemetryBuilder(componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)

	cfg := &Config{
		CheckInterval:         time.Second,
		MemoryLimitPercentage: 99,
		MemorySpikePercentage: 99,
	}

	ml, err := newMemoryLimiter(cfg, zap.NewNop(), tb)
	require.NoError(t, err)

	called := false
	handler := func(_ context.Context, req any) (any, error) {
		called = true
		return req, nil
	}

	resp, err := ml.grpcUnaryInterceptor(ctx, "request", nil, handler)

	require.NoError(t, err)
	assert.Equal(t, "request", resp)
	assert.True(t, called)
}

func TestGRPCUnaryInterceptor_Refused(t *testing.T) {
	ctx := context.Background()

	tb, err := metadata.NewTelemetryBuilder(componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)

	ml := newRefusingMemoryLimiter(t, tb)

	called := false
	handler := func(_ context.Context, _ any) (any, error) {
		called = true
		return "success", nil
	}

	resp, err := ml.grpcUnaryInterceptor(ctx, "request", nil, handler)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.ResourceExhausted, status.Code(err))
	assert.False(t, called)
}

func TestGRPCStreamInterceptor_Normal(t *testing.T) {
	ctx := context.Background()

	tb, err := metadata.NewTelemetryBuilder(componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)

	cfg := &Config{
		CheckInterval:         time.Second,
		MemoryLimitPercentage: 99,
		MemorySpikePercentage: 99,
	}

	ml, err := newMemoryLimiter(cfg, zap.NewNop(), tb)
	require.NoError(t, err)

	called := false
	handler := func(_ any, _ grpc.ServerStream) error {
		called = true
		return nil
	}

	stream := &mockServerStream{ctx: ctx}

	err = ml.grpcStreamInterceptor(nil, stream, nil, handler)

	require.NoError(t, err)
	assert.True(t, called)
}

func TestGRPCStreamInterceptor_Refused(t *testing.T) {
	ctx := context.Background()

	tb, err := metadata.NewTelemetryBuilder(componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)

	ml := newRefusingMemoryLimiter(t, tb)

	called := false
	handler := func(_ any, _ grpc.ServerStream) error {
		called = true
		return nil
	}

	stream := &mockServerStream{ctx: ctx}

	err = ml.grpcStreamInterceptor(nil, stream, nil, handler)

	require.Error(t, err)
	assert.Equal(t, codes.ResourceExhausted, status.Code(err))
	assert.False(t, called)
}

func TestWrapHTTPHandler_Normal(t *testing.T) {
	ctx := context.Background()

	tb, err := metadata.NewTelemetryBuilder(componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)

	cfg := &Config{
		CheckInterval:         time.Second,
		MemoryLimitPercentage: 99,
		MemorySpikePercentage: 99,
	}

	ml, err := newMemoryLimiter(cfg, zap.NewNop(), tb)
	require.NoError(t, err)

	called := false
	base := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler, err := ml.wrapHTTPHandler(ctx, base)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/", http.NoBody)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestWrapHTTPHandler_Refused(t *testing.T) {
	ctx := context.Background()

	testTel := componenttest.NewTelemetry()

	tb, err := metadata.NewTelemetryBuilder(testTel.NewTelemetrySettings())
	require.NoError(t, err)
	defer tb.Shutdown()

	ml := newRefusingMemoryLimiter(t, tb)

	called := false
	base := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	})

	wrap, err := ml.GetHTTPHandler(ctx)
	require.NoError(t, err)

	handler, err := wrap(ctx, base)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/", http.NoBody)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.False(t, called)

	metadatatest.AssertEqualMemorylimiterRefusedRequests(
		t,
		testTel,
		[]metricdata.DataPoint[int64]{
			{
				Value: 1,
				Attributes: attribute.NewSet(
					attribute.String("transport", "http"),
				),
			},
		},
		metricdatatest.IgnoreTimestamp(),
	)
}

func TestGetGRPCServerOptions(t *testing.T) {
	tb, err := metadata.NewTelemetryBuilder(componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)

	cfg := &Config{
		CheckInterval:         time.Second,
		MemoryLimitPercentage: 99,
		MemorySpikePercentage: 99,
	}

	ml, err := newMemoryLimiter(cfg, zap.NewNop(), tb)
	require.NoError(t, err)

	opts, err := ml.GetGRPCServerOptions(context.Background())

	require.NoError(t, err)
	require.Len(t, opts, 2)
}
