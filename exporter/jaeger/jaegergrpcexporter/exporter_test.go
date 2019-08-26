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

package jaegergrpcexporter

import (
	"context"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-service/config/configmodels"
	"github.com/open-telemetry/opentelemetry-service/consumer/consumerdata"
)

func TestNew(t *testing.T) {
	type args struct {
		exporterName      string
		collectorEndpoint string
		secure            bool
		certPemFile       string
		serverOverride    string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "empty_exporterName",
			args: args{
				collectorEndpoint: "127.0.0.1:55678",
			},
			wantErr: true,
		},
		{
			name: "createExporter",
			args: args{
				exporterName:      typeStr,
				collectorEndpoint: "some.non.existent:55678",
			},
		},
		{
			name: "createSecureExporter",
			args: args{
				collectorEndpoint: "foo:55",
				exporterName:      typeStr,
				secure:            true,
			},
		},
		{
			name: "createSecureExporterWithClientTLS",
			args: args{
				collectorEndpoint: "foo:55",
				exporterName:      typeStr,
				secure:            true,
				certPemFile:       path.Join(".", "testdata", "test_cert.pem"),
				serverOverride:    "foo",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Endpoint: tt.args.collectorEndpoint,
				ExporterSettings: configmodels.ExporterSettings{
					NameVal:        tt.args.exporterName,
					UseSecure:      tt.args.secure,
					CertPemFile:    tt.args.certPemFile,
					ServerOverride: tt.args.serverOverride,
				},
			}
			got, err := New(*config)
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
