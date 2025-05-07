// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ptrace // import "go.opentelemetry.io/collector/pdata/ptrace"

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStatusCodeString(t *testing.T) {
	assert.Equal(t, "Unset", StatusCodeUnset.String())
	assert.Equal(t, "Ok", StatusCodeOk.String())
	assert.Equal(t, "Error", StatusCodeError.String())
	assert.Empty(t, StatusCode(100).String())
}
