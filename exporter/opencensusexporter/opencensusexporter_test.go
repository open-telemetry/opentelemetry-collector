// Copyright 2019, OpenCensus Authors
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
	"testing"

	"github.com/spf13/viper"
)

func TestOpenCensusTraceExportersFromViper(t *testing.T) {
	v := viper.New()
	v.Set("opencensus", map[interface{}]interface{}{})
	v.Set("opencensus.endpoint", "")
	_, _, _, err := OpenCensusTraceExportersFromViper(v)

	if err != ErrEndpointRequired {
		t.Fatalf("Expected to get ErrEndpointRequired. Got %v", err)
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

	if err != ErrUnableToGetTLSCreds {
		t.Fatalf("Expected to get ErrUnableToGetTLSCreds but did not")
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
	if err != ErrUnsupportedCompressionType {
		t.Fatalf("Expected to get ErrUnsupportedCompressionType but did not")
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
