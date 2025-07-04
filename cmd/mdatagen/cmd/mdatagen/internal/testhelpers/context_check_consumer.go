package testhelpers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

const (
	TestKey   = "ctx-prop-key"
	TestValue = "expected-value"
)

type ContextCheckLogsConsumer struct {
	T *testing.T
}

func (c *ContextCheckLogsConsumer) ConsumeLogs(ctx context.Context, logs plog.Logs) error {
	val := ctx.Value(TestKey)
	require.Equal(c.T, TestValue, val)
	return nil
}

type ContextCheckMetricsConsumer struct {
	T *testing.T
}

func (c *ContextCheckMetricsConsumer) ConsumeMetrics(ctx context.Context, metrics pmetric.Metrics) error {
	val := ctx.Value(TestKey)
	require.Equal(c.T, TestValue, val)
	return nil
}

type ContextCheckTracesConsumer struct {
	T *testing.T
}

func (c *ContextCheckTracesConsumer) ConsumeTraces(ctx context.Context, traces ptrace.Traces) error {
	val := ctx.Value(TestKey)
	require.Equal(c.T, TestValue, val)
	return nil
}
