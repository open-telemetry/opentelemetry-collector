// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ptrace // import "go.opentelemetry.io/collector/pdata/ptrace"

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStatusCodeString(t *testing.T) {
	assert.EqualValues(t, "Unset", StatusCodeUnset.String())
	assert.EqualValues(t, "Ok", StatusCodeOk.String())
	assert.EqualValues(t, "Error", StatusCodeError.String())
	assert.EqualValues(t, "", StatusCode(100).String())
}
