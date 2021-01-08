// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testdata

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal"
	otlplogs "go.opentelemetry.io/collector/internal/data/protogen/logs/v1"
)

type logTestCase struct {
	name string
	ld   pdata.Logs
	otlp []*otlplogs.ResourceLogs
}

func generateAllLogTestCases() []logTestCase {
	return []logTestCase{
		{
			name: "empty",
			ld:   GenerateLogDataEmpty(),
			otlp: generateLogOtlpEmpty(),
		},
		{
			name: "one-empty-resource-logs",
			ld:   GenerateLogDataOneEmptyResourceLogs(),
			otlp: generateLogOtlpOneEmptyResourceLogs(),
		},
		{
			name: "no-log-records",
			ld:   GenerateLogDataNoLogRecords(),
			otlp: generateLogOtlpNoLogRecords(),
		},
		{
			name: "one-empty-log-record",
			ld:   GenerateLogDataOneEmptyLogs(),
			otlp: generateLogOtlpOneEmptyLogs(),
		},
		{
			name: "one-log-record-no-resource",
			ld:   GenerateLogDataOneLogNoResource(),
			otlp: generateLogOtlpOneLogNoResource(),
		},
		{
			name: "one-log-record",
			ld:   GenerateLogDataOneLog(),
			otlp: generateLogOtlpOneLog(),
		},
		{
			name: "two-records-same-resource",
			ld:   GenerateLogDataTwoLogsSameResource(),
			otlp: GenerateLogOtlpSameResourceTwoLogs(),
		},
		{
			name: "two-records-same-resource-one-different",
			ld:   GenerateLogDataTwoLogsSameResourceOneDifferent(),
			otlp: generateLogOtlpTwoLogsSameResourceOneDifferent(),
		},
	}
}

func TestToFromOtlpLog(t *testing.T) {
	allTestCases := generateAllLogTestCases()
	// Ensure NumLogTests gets updated.
	for i := range allTestCases {
		test := allTestCases[i]
		t.Run(test.name, func(t *testing.T) {
			ld := pdata.LogsFromInternalRep(internal.LogsFromOtlp(test.otlp))
			assert.EqualValues(t, test.ld, ld)
			otlp := internal.LogsToOtlp(ld.InternalRep())
			assert.EqualValues(t, test.otlp, otlp)
		})
	}
}
