// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ptrace // import "go.opentelemetry.io/collector/pdata/ptrace"

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSpanKindString(t *testing.T) {
	assert.EqualValues(t, "Unspecified", SpanKindUnspecified.String())
	assert.EqualValues(t, "Internal", SpanKindInternal.String())
	assert.EqualValues(t, "Server", SpanKindServer.String())
	assert.EqualValues(t, "Client", SpanKindClient.String())
	assert.EqualValues(t, "Producer", SpanKindProducer.String())
	assert.EqualValues(t, "Consumer", SpanKindConsumer.String())
	assert.EqualValues(t, "", SpanKind(100).String())
}
