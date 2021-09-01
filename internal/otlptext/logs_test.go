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

package otlptext

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/model/pdata"
)

func TestLogsText(t *testing.T) {
	type args struct {
		ld pdata.Logs
	}
	tests := []struct {
		name  string
		args  args
		empty bool
	}{
		{"empty logs", args{pdata.NewLogs()}, true},
		{"logs data with empty resource log", args{testdata.GenerateLogsOneEmptyResourceLogs()}, false},
		{"logs data with no log records", args{testdata.GenerateLogsNoLogRecords()}, false},
		{"logs with one empty log", args{testdata.GenerateLogsOneEmptyLogRecord()}, false},
		{"logs with one log", args{testdata.GenerateLogsOneLogRecord()}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logs, err := NewTextLogsMarshaler().MarshalLogs(tt.args.ld)
			assert.NoError(t, err)
			if !tt.empty {
				assert.NotEmpty(t, logs)
			}
		})
	}
}
