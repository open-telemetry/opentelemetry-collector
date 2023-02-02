// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by "pdata/internal/cmd/pdatagen/main.go". DO NOT EDIT.
// To regenerate this file run "make genpdata".

package pmetric

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/pdata/internal"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestNumberDataPoint_MoveTo(t *testing.T) {
	ms := generateTestNumberDataPoint()
	dest := NewNumberDataPoint()
	ms.MoveTo(dest)
	assert.Equal(t, NewNumberDataPoint(), ms)
	assert.Equal(t, generateTestNumberDataPoint(), dest)
}

func TestNumberDataPoint_CopyTo(t *testing.T) {
	ms := NewNumberDataPoint()
	orig := NewNumberDataPoint()
	orig.CopyTo(ms)
	assert.Equal(t, orig, ms)
	orig = generateTestNumberDataPoint()
	orig.CopyTo(ms)
	assert.Equal(t, orig, ms)
}

func TestNumberDataPoint_Attributes(t *testing.T) {
	ms := NewNumberDataPoint()
	assert.Equal(t, pcommonNewMap(), ms.Attributes())
	internal.FillTestMap(internal.Map(ms.Attributes()))
	assert.Equal(t, pcommon.Map(internal.GenerateTestMap()), ms.Attributes())
}

func TestNumberDataPoint_StartTimestamp(t *testing.T) {
	ms := NewNumberDataPoint()
	assert.Equal(t, pcommon.Timestamp(0), ms.StartTimestamp())
	testValStartTimestamp := pcommon.Timestamp(1234567890)
	ms.SetStartTimestamp(testValStartTimestamp)
	assert.Equal(t, testValStartTimestamp, ms.StartTimestamp())
}

func TestNumberDataPoint_Timestamp(t *testing.T) {
	ms := NewNumberDataPoint()
	assert.Equal(t, pcommon.Timestamp(0), ms.Timestamp())
	testValTimestamp := pcommon.Timestamp(1234567890)
	ms.SetTimestamp(testValTimestamp)
	assert.Equal(t, testValTimestamp, ms.Timestamp())
}

func TestNumberDataPoint_ValueType(t *testing.T) {
	tv := NewNumberDataPoint()
	assert.Equal(t, NumberDataPointValueTypeEmpty, tv.ValueType())
}

func TestNumberDataPoint_DoubleValue(t *testing.T) {
	ms := NewNumberDataPoint()
	assert.Equal(t, float64(0.0), ms.DoubleValue())
	ms.SetDoubleValue(float64(17.13))
	assert.Equal(t, float64(17.13), ms.DoubleValue())
	assert.Equal(t, NumberDataPointValueTypeDouble, ms.ValueType())
}

func TestNumberDataPoint_IntValue(t *testing.T) {
	ms := NewNumberDataPoint()
	assert.Equal(t, int64(0), ms.IntValue())
	ms.SetIntValue(int64(17))
	assert.Equal(t, int64(17), ms.IntValue())
	assert.Equal(t, NumberDataPointValueTypeInt, ms.ValueType())
}

func TestNumberDataPoint_Exemplars(t *testing.T) {
	ms := NewNumberDataPoint()
	assert.Equal(t, NewExemplarSlice(), ms.Exemplars())
	fillTestExemplarSlice(ms.Exemplars())
	assert.Equal(t, generateTestExemplarSlice(), ms.Exemplars())
}

func TestNumberDataPoint_Flags(t *testing.T) {
	ms := NewNumberDataPoint()
	assert.Equal(t, DataPointFlags(0), ms.Flags())
	testValFlags := DataPointFlags(1)
	ms.SetFlags(testValFlags)
	assert.Equal(t, testValFlags, ms.Flags())
}
