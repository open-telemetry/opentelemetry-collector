package clogtest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/testdata"
)

func TestLogsSink(t *testing.T) {
	sink := new(LogsSink)
	md := testdata.GenerateLogs(1)
	want := make([]plog.Logs, 0, 7)
	for i := 0; i < 7; i++ {
		require.NoError(t, sink.ConsumeLogs(context.Background(), md))
		want = append(want, md)
	}
	assert.Equal(t, want, sink.AllLogs())
	assert.Equal(t, len(want), sink.LogRecordCount())
	sink.Reset()
	assert.Equal(t, 0, len(sink.AllLogs()))
	assert.Equal(t, 0, sink.LogRecordCount())
}
