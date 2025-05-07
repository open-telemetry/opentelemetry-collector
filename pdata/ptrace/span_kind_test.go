// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ptrace // import "go.opentelemetry.io/collector/pdata/ptrace"

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSpanKindString(t *testing.T) {
	assert.Equal(t, "Unspecified", SpanKindUnspecified.String())
	assert.Equal(t, "Internal", SpanKindInternal.String())
	assert.Equal(t, "Server", SpanKindServer.String())
	assert.Equal(t, "Client", SpanKindClient.String())
	assert.Equal(t, "Producer", SpanKindProducer.String())
	assert.Equal(t, "Consumer", SpanKindConsumer.String())
	assert.Empty(t, SpanKind(100).String())
}
