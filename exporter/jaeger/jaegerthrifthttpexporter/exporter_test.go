// Copyright 2019, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package jaegerthrifthttpexporter

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
)

func TestNew(t *testing.T) {
	const testHTTPAddress = "http://a.test.dom:123/at/some/path"

	type args struct {
		config      configmodels.Exporter
		httpAddress string
		headers     map[string]string
		timeout     time.Duration
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "empty_exporterName",
			args: args{
				config:      nil,
				httpAddress: testHTTPAddress,
			},
			wantErr: true,
		},
		{
			name: "createExporter",
			args: args{
				config:      &configmodels.ExporterSettings{},
				httpAddress: testHTTPAddress,
				headers:     map[string]string{"test": "test"},
				timeout:     10 * time.Nanosecond,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.args.config, tt.args.httpAddress, tt.args.headers, tt.args.timeout)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got == nil {
				return
			}

			// This is expected to fail.
			err = got.ConsumeTraceData(context.Background(), consumerdata.TraceData{})
			assert.Error(t, err)
		})
	}
}
