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

package ptrace

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/pdata/internal"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestSpanEvent_MoveTo(t *testing.T) {
	ms := generateTestSpanEvent()
	dest := NewSpanEvent()
	ms.MoveTo(dest)
	assert.Equal(t, NewSpanEvent(), ms)
	assert.Equal(t, generateTestSpanEvent(), dest)
}

func TestSpanEvent_CopyTo(t *testing.T) {
	ms := NewSpanEvent()
	orig := NewSpanEvent()
	orig.CopyTo(ms)
	assert.Equal(t, orig, ms)
	orig = generateTestSpanEvent()
	orig.CopyTo(ms)
	assert.Equal(t, orig, ms)
}

func TestSpanEvent_Timestamp(t *testing.T) {
	ms := NewSpanEvent()
	assert.Equal(t, pcommon.Timestamp(0), ms.Timestamp())
	testValTimestamp := pcommon.Timestamp(1234567890)
	ms.SetTimestamp(testValTimestamp)
	assert.Equal(t, testValTimestamp, ms.Timestamp())
}

func TestSpanEvent_Name(t *testing.T) {
	ms := NewSpanEvent()
	assert.Equal(t, "", ms.Name())
	ms.SetName("test_name")
	assert.Equal(t, "test_name", ms.Name())
}

func TestSpanEvent_Attributes(t *testing.T) {
	ms := NewSpanEvent()
	assert.Equal(t, pcommonNewMap(), ms.Attributes())
	internal.FillTestMap(internal.Map(ms.Attributes()))
	assert.Equal(t, pcommon.Map(internal.GenerateTestMap()), ms.Attributes())
}

func TestSpanEvent_DroppedAttributesCount(t *testing.T) {
	ms := NewSpanEvent()
	assert.Equal(t, uint32(0), ms.DroppedAttributesCount())
	ms.SetDroppedAttributesCount(uint32(17))
	assert.Equal(t, uint32(17), ms.DroppedAttributesCount())
}
