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
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/service/telemetry"
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
			err:          fmt.Errorf("invalid args for otlp exporter: <nil>"),
		},
		{
			name:         "otlp grpc exporter with no args",
			exporterType: "otlp",
			args: map[string]any{
				"protocol": "grpc/protobuf",
			},
		},
		{
			name:         "otlp grpc exporter with invalid args",
			exporterType: "otlp",
			args:         "string arg",
			err:          fmt.Errorf("invalid args for otlp exporter: string arg"),
		},
		{
			name:         "otlp grpc exporter with invalid timeout",
			exporterType: "otlp",
			args: map[string]any{
				"protocol": "grpc/protobuf",
				"timeout":  "string arg",
			},
			err: fmt.Errorf("invalid timeout for otlp exporter: string arg"),
		},
		{
			name:         "otlp grpc exporter with invalid headers",
			exporterType: "otlp",
			args: map[string]any{
				"protocol": "grpc/protobuf",
				"headers":  "string arg",
			},
			err: fmt.Errorf("invalid headers for otlp exporter: string arg"),
		},
		{
			name:         "otlp grpc exporter with valid options",
			exporterType: "otlp",
			args: map[string]any{
				"protocol": "grpc/protobuf",
				"headers": map[string]any{
					"header-key": "header-val",
				},
				"timeout":     100,
				"endpoint":    "localhost:4318",
				"compression": "gzip",
			},
		},
		{
			name:         "otlp http exporter with valid options",
			exporterType: "otlp",
			args: map[string]any{
				"protocol": "http/protobuf",
				"headers": map[string]any{
					"header-key": "header-val",
				},
				"timeout":     100,
				"endpoint":    "localhost:4318",
				"compression": "gzip",
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

func TestInitPeriodicReader(t *testing.T) {
	testCases := []struct {
		name   string
		reader telemetry.MetricReader
		args   any
		err    error
	}{
		{
			name: "no exporter",
			err:  errors.New("no exporter configured"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := InitPeriodicReader(context.Background(), tc.reader)
			assert.Equal(t, tc.err, err)
		})
	}
}
