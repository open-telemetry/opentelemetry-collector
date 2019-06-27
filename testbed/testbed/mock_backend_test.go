package testbed

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func TestGeneratorAndBackend(t *testing.T) {
	mb := NewMockBackend("mockbackend.log")

	assert.EqualValues(t, 0, mb.SpansReceived())

	err := mb.Start(BackendJaeger)
	require.NoError(t, err, "Cannot start backend")

	defer mb.Stop()

	lg, err := NewLoadGenerator()
	require.NoError(t, err, "Cannot start load generator")

	assert.EqualValues(t, 0, lg.SpansSent)

	// Generate for about 10ms at 1000 SPS
	lg.Start(LoadOptions{SpansPerSecond: 1000})

	time.Sleep(time.Millisecond * 10)

	lg.Stop()

	// Presumably should have generated something. If not then the testbed is very slow
	// so we will consider it a failure.
	assert.True(t, lg.SpansSent > 0)

	// The backend should receive everything generated.
	assert.Equal(t, lg.SpansSent, mb.SpansReceived())
}
