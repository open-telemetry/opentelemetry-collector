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

package plog

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/pdata/internal"
)

func TestScopeLogs_MoveTo(t *testing.T) {
	ms := generateTestScopeLogs()
	dest := NewScopeLogs()
	ms.MoveTo(dest)
	assert.Equal(t, NewScopeLogs(), ms)
	assert.Equal(t, generateTestScopeLogs(), dest)
}

func TestScopeLogs_CopyTo(t *testing.T) {
	ms := NewScopeLogs()
	orig := NewScopeLogs()
	orig.CopyTo(ms)
	assert.Equal(t, orig, ms)
	orig = generateTestScopeLogs()
	orig.CopyTo(ms)
	assert.Equal(t, orig, ms)
}

func TestScopeLogs_Scope(t *testing.T) {
	ms := NewScopeLogs()
	internal.FillTestInstrumentationScope(internal.InstrumentationScope(ms.Scope()))
	assert.Equal(t, internal.GenerateTestInstrumentationScope().GetOrig(), internal.InstrumentationScope(ms.Scope()).GetOrig())
}

func TestScopeLogs_SchemaUrl(t *testing.T) {
	ms := NewScopeLogs()
	assert.Equal(t, "", ms.SchemaUrl())
	ms.SetSchemaUrl("https://opentelemetry.io/schemas/1.5.0")
	assert.Equal(t, "https://opentelemetry.io/schemas/1.5.0", ms.SchemaUrl())
}

func TestScopeLogs_LogRecords(t *testing.T) {
	ms := NewScopeLogs()
	assert.Equal(t, NewLogRecordSlice().getOrig(), ms.LogRecords().getOrig())
	fillTestLogRecordSlice(ms.LogRecords())
	assert.Equal(t, generateTestLogRecordSlice().getOrig(), ms.LogRecords().getOrig())
}

func generateTestScopeLogs() ScopeLogs {
	tv := NewScopeLogs()
	fillTestScopeLogs(tv)
	return tv
}

func fillTestScopeLogs(tv ScopeLogs) {
	internal.FillTestInstrumentationScope(internal.NewInstrumentationScope(&tv.getOrig().Scope,
		wrappedScopeLogsScope{ScopeLogs: tv}))
	tv.getOrig().SchemaUrl = "https://opentelemetry.io/schemas/1.5.0"
	fillTestLogRecordSlice(newLogRecordSlice(&tv.getOrig().LogRecords, tv))
}
