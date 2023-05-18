// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package plog // import "go.opentelemetry.io/collector/pdata/plog"

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSeverityNumberString(t *testing.T) {
	assert.EqualValues(t, "Unspecified", SeverityNumberUnspecified.String())
	assert.EqualValues(t, "Trace", SeverityNumberTrace.String())
	assert.EqualValues(t, "Trace2", SeverityNumberTrace2.String())
	assert.EqualValues(t, "Trace3", SeverityNumberTrace3.String())
	assert.EqualValues(t, "Trace4", SeverityNumberTrace4.String())
	assert.EqualValues(t, "Debug", SeverityNumberDebug.String())
	assert.EqualValues(t, "Debug2", SeverityNumberDebug2.String())
	assert.EqualValues(t, "Debug3", SeverityNumberDebug3.String())
	assert.EqualValues(t, "Debug4", SeverityNumberDebug4.String())
	assert.EqualValues(t, "Info", SeverityNumberInfo.String())
	assert.EqualValues(t, "Info2", SeverityNumberInfo2.String())
	assert.EqualValues(t, "Info3", SeverityNumberInfo3.String())
	assert.EqualValues(t, "Info4", SeverityNumberInfo4.String())
	assert.EqualValues(t, "Warn", SeverityNumberWarn.String())
	assert.EqualValues(t, "Warn2", SeverityNumberWarn2.String())
	assert.EqualValues(t, "Warn3", SeverityNumberWarn3.String())
	assert.EqualValues(t, "Warn4", SeverityNumberWarn4.String())
	assert.EqualValues(t, "Error", SeverityNumberError.String())
	assert.EqualValues(t, "Error2", SeverityNumberError2.String())
	assert.EqualValues(t, "Error3", SeverityNumberError3.String())
	assert.EqualValues(t, "Error4", SeverityNumberError4.String())
	assert.EqualValues(t, "Fatal", SeverityNumberFatal.String())
	assert.EqualValues(t, "Fatal2", SeverityNumberFatal2.String())
	assert.EqualValues(t, "Fatal3", SeverityNumberFatal3.String())
	assert.EqualValues(t, "Fatal4", SeverityNumberFatal4.String())
	assert.EqualValues(t, "", SeverityNumber(100).String())
}
