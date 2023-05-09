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

package proctelemetry // import "go.opentelemetry.io/collector/service/internal/proctelemetry"

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitExporter(t *testing.T) {
	testCases := []struct {
		name         string
		exporterType string
		args         any
		err          error
	}{
		{
			name:         "unsupported exporter",
			exporterType: "unsupported",
			err:          fmt.Errorf("unsupported metric exporter type: unsupported"),
		},
		{
			name:         "console exporter",
			exporterType: "console",
		},
		{
			name:         "otlp exporter with no args",
			exporterType: "otlp",
		},
		{
			name:         "otlp exporter with invalid args",
			exporterType: "otlp",
			args:         "string arg",
			err:          fmt.Errorf("invalid args for otlp exporter: string arg"),
		},
		{
			name:         "otlp exporter with invalid timeout",
			exporterType: "otlp",
			args: map[string]interface{}{
				"timeout": "string arg",
			},
			err: fmt.Errorf("invalid timeout for otlp exporter: string arg"),
		},
		{
			name:         "otlp exporter with invalid headers",
			exporterType: "otlp",
			args: map[string]interface{}{
				"headers": "string arg",
			},
			err: fmt.Errorf("invalid headers for otlp exporter: string arg"),
		},
		{
			name:         "otlp exporter with valid options",
			exporterType: "otlp",
			args: map[string]interface{}{
				"headers": map[string]any{
					"header-key": "header-val",
				},
				"timeout": 100,
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := InitExporter(context.Background(), tc.exporterType, tc.args)
			assert.Equal(t, tc.err, err)
		})
	}
}
