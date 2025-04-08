// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package plog // import "go.opentelemetry.io/collector/pdata/plog"

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSeverityNumberString(t *testing.T) {
	assert.Equal(t, "Unspecified", SeverityNumberUnspecified.String())
	assert.Equal(t, "Trace", SeverityNumberTrace.String())
	assert.Equal(t, "Trace2", SeverityNumberTrace2.String())
	assert.Equal(t, "Trace3", SeverityNumberTrace3.String())
	assert.Equal(t, "Trace4", SeverityNumberTrace4.String())
	assert.Equal(t, "Debug", SeverityNumberDebug.String())
	assert.Equal(t, "Debug2", SeverityNumberDebug2.String())
	assert.Equal(t, "Debug3", SeverityNumberDebug3.String())
	assert.Equal(t, "Debug4", SeverityNumberDebug4.String())
	assert.Equal(t, "Info", SeverityNumberInfo.String())
	assert.Equal(t, "Info2", SeverityNumberInfo2.String())
	assert.Equal(t, "Info3", SeverityNumberInfo3.String())
	assert.Equal(t, "Info4", SeverityNumberInfo4.String())
	assert.Equal(t, "Warn", SeverityNumberWarn.String())
	assert.Equal(t, "Warn2", SeverityNumberWarn2.String())
	assert.Equal(t, "Warn3", SeverityNumberWarn3.String())
	assert.Equal(t, "Warn4", SeverityNumberWarn4.String())
	assert.Equal(t, "Error", SeverityNumberError.String())
	assert.Equal(t, "Error2", SeverityNumberError2.String())
	assert.Equal(t, "Error3", SeverityNumberError3.String())
	assert.Equal(t, "Error4", SeverityNumberError4.String())
	assert.Equal(t, "Fatal", SeverityNumberFatal.String())
	assert.Equal(t, "Fatal2", SeverityNumberFatal2.String())
	assert.Equal(t, "Fatal3", SeverityNumberFatal3.String())
	assert.Equal(t, "Fatal4", SeverityNumberFatal4.String())
	assert.Empty(t, SeverityNumber(100).String())
}
