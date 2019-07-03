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

package opencensusexporter

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/spf13/viper"

	"github.com/open-telemetry/opentelemetry-service/consumer/consumerdata"
)

func TestOpenCensusTraceExportersFromViper(t *testing.T) {
	v := viper.New()
	v.Set("opencensus", map[interface{}]interface{}{})
	v.Set("opencensus.endpoint", "")
	_, _, _, err := OpenCensusTraceExportersFromViper(v)

	if errorCode(err) != errEndpointRequired {
		t.Fatalf("Expected to get errEndpointRequired. Got %v", err)
	}

	v.Set("opencensus.endpoint", "127.0.0.1:55678")
	exporters, _, _, err := OpenCensusTraceExportersFromViper(v)

	if err != nil {
		t.Fatalf("Unexpected error building OpenCensus Exporter")
	}
	if len(exporters) != 1 {
		t.Fatalf("Should get 1 exporter but got %d", len(exporters))
	}
}

func TestOpenCensusTraceExportersFromViper_TLS(t *testing.T) {
	v := viper.New()
	v.Set("opencensus.endpoint", "127.0.0.1:55678")
	v.Set("opencensus.cert-pem-file", "dummy_file.pem")
	_, _, _, err := OpenCensusTraceExportersFromViper(v)

	if errorCode(err) != errUnableToGetTLSCreds {
		t.Fatalf("Expected to get errUnableToGetTLSCreds but got %v", err)
	}

	v.Set("opencensus.cert-pem-file", "testdata/test_cert.pem")
	exporters, _, _, err := OpenCensusTraceExportersFromViper(v)
	if err != nil {
		t.Fatalf("Unexpected error building OpenCensus Exporter")
	}
	if len(exporters) != 1 {
		t.Fatalf("Should get 1 exporter but got %d", len(exporters))
	}
}

func TestOpenCensusTraceExportersFromViper_Compression(t *testing.T) {
	v := viper.New()
	v.Set("opencensus.endpoint", "127.0.0.1:55678")
	v.Set("opencensus.compression", "random-compression")
	_, _, _, err := OpenCensusTraceExportersFromViper(v)
	if errorCode(err) != errUnsupportedCompressionType {
		t.Fatalf("Expected to get errUnsupportedCompressionType but got %v", err)
	}

	v.Set("opencensus.compression", "gzip")
	exporters, _, _, err := OpenCensusTraceExportersFromViper(v)
	if err != nil {
		t.Fatalf("Unexpected error building OpenCensus Exporter")
	}
	if len(exporters) != 1 {
		t.Fatalf("Should get 1 exporter but got %d", len(exporters))
	}
}

func TestOpenCensusTraceExporters_StopError(t *testing.T) {
	v := viper.New()
	v.Set("opencensus.endpoint", "127.0.0.1:55678")
	tps, _, doneFns, err := OpenCensusTraceExportersFromViper(v)
	doneFnCalled := false
	defer func() {
		if doneFnCalled {
			return
		}
		for _, doneFn := range doneFns {
			doneFn()
		}
	}()
	if err != nil {
		t.Fatalf("got = %v, want = nil", err)
	}
	if len(tps) != 1 {
		t.Fatalf("got %d trace exporters, want 1", len(tps))
	}
	if len(doneFns) != 1 {
		t.Fatalf("got %d close functions, want 1", len(doneFns))
	}

	err = doneFns[0]()
	doneFnCalled = true
	if err != nil {
		t.Fatalf("doneFn[0]() got = %v, want = nil", err)
	}
	err = tps[0].ConsumeTraceData(context.Background(), consumerdata.TraceData{})
	if errorCode(err) != errAlreadyStopped {
		t.Fatalf("Expected to get errAlreadyStopped but got %v", err)
	}
}

// TestOpenCensusTraceExporterConfigsViaViper validates with specific parts of the configuration
// are correctly read via Viper. It ensures that instances can be created with that config, however,
// it doesn't validate that the settings were actually applied (that requires changes to better expose
// internal data from the exporter or use of reflection).
func TestOpenCensusTraceExporterConfigsViaViper(t *testing.T) {
	const defaultTestEndPoint = "127.0.0.1:55678"
	tests := []struct {
		name      string
		configMap map[string]string
		want      opencensusConfig
	}{
		{
			name: "UseSecure",
			configMap: map[string]string{
				"opencensus.endpoint": defaultTestEndPoint,
				"opencensus.secure":   "true",
			},
			want: opencensusConfig{
				Endpoint:  defaultTestEndPoint,
				UseSecure: true,
			},
		},
		{
			name: "ReconnectionDelay",
			configMap: map[string]string{
				"opencensus.endpoint":           defaultTestEndPoint,
				"opencensus.reconnection-delay": "5s",
			},
			want: opencensusConfig{
				Endpoint:          defaultTestEndPoint,
				ReconnectionDelay: 5 * time.Second,
			},
		},
		{
			name: "KeepaliveParameters",
			configMap: map[string]string{
				"opencensus.endpoint":                        defaultTestEndPoint,
				"opencensus.keepalive.time":                  "30s",
				"opencensus.keepalive.timeout":               "25s",
				"opencensus.keepalive.permit-without-stream": "true",
			},
			want: opencensusConfig{
				Endpoint: defaultTestEndPoint,
				KeepaliveParameters: &keepaliveConfig{
					Time:                30 * time.Second,
					Timeout:             25 * time.Second,
					PermitWithoutStream: true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := viper.New()
			for key, value := range tt.configMap {
				v.Set(key, value)
			}

			// Ensure that the settings are being read, via UnmarshalExact.
			var got opencensusConfig
			if err := v.Sub("opencensus").UnmarshalExact(&got); err != nil {
				t.Fatalf("UnmarshalExact() error: %v", err)
			}

			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("Mismatched configs\n-Got +Want:\n\t%s", diff)
			}

			// Ensure creation happens as expected.
			tps, _, doneFns, err := OpenCensusTraceExportersFromViper(v)
			defer func() {
				for _, doneFn := range doneFns {
					doneFn()
				}
			}()
			if err != nil {
				t.Fatalf("got = %v, want = nil", err)
			}
			if len(tps) != 1 {
				t.Fatalf("got %d trace exporters, want 1", len(tps))
			}
			if len(doneFns) != 1 {
				t.Fatalf("got %d close functions, want 1", len(doneFns))
			}
		})
	}
}

func errorCode(err error) ocTraceExporterErrorCode {
	ocErr, ok := err.(*ocTraceExporterError)
	if !ok {
		return ocTraceExporterErrorCode(0)
	}
	return ocErr.code
}
