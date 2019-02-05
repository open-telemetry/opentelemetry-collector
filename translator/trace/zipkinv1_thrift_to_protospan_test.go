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

package tracetranslator

import (
	"encoding/json"
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/jaegertracing/jaeger/thrift-gen/zipkincore"
)

func TestZipkinV1ThriftToOCProto(t *testing.T) {
	blob, err := ioutil.ReadFile("./testdata/zipkin_v1_thrift_single_batch.json")
	if err != nil {
		t.Fatalf("failed to load test data: %v", err)
	}

	var ztSpans []*zipkincore.Span
	err = json.Unmarshal(blob, &ztSpans)
	if err != nil {
		t.Fatalf("failed to unmarshal json into zipkin v1 thrift: %v", err)
	}

	got, err := ZipkinV1ThriftBatchToOCProto(ztSpans)
	if err != nil {
		t.Fatalf("failed to translate zipkinv1 thrift to OC proto: %v", err)
	}

	want := ocBatchesFromZipkinV1
	sortTraceByNodeName(want)
	sortTraceByNodeName(got)

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got different data than want")
	}
}

func BenchmarkZipkinV1ThriftToOCProto(b *testing.B) {
	blob, err := ioutil.ReadFile("./testdata/zipkin_v1_thrift_single_batch.json")
	if err != nil {
		b.Fatalf("failed to load test data: %v", err)
	}

	var ztSpans []*zipkincore.Span
	err = json.Unmarshal(blob, &ztSpans)
	if err != nil {
		b.Fatalf("failed to unmarshal json into zipkin v1 thrift: %v", err)
	}

	for n := 0; n < b.N; n++ {
		ZipkinV1ThriftBatchToOCProto(ztSpans)
	}
}

func Test_bytesInt16ToInt64(t *testing.T) {
	tests := []struct {
		name    string
		bytes   []byte
		want    int64
		wantErr error
	}{
		{
			name:    "too short byte slice",
			bytes:   nil,
			want:    0,
			wantErr: errNotEnoughBytes,
		},
		{
			name:    "exact size byte slice",
			bytes:   []byte{0, 200},
			want:    200,
			wantErr: nil,
		},
		{
			name:    "large byte slice",
			bytes:   []byte{0, 128, 200, 200},
			want:    128,
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := bytesInt16ToInt64(tt.bytes)
			if err != tt.wantErr {
				t.Errorf("bytesInt16ToInt64() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("bytesInt16ToInt64() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_bytesInt32ToInt64(t *testing.T) {
	tests := []struct {
		name    string
		bytes   []byte
		want    int64
		wantErr error
	}{
		{
			name:    "too short byte slice",
			bytes:   []byte{},
			want:    0,
			wantErr: errNotEnoughBytes,
		},
		{
			name:    "exact size byte slice",
			bytes:   []byte{0, 0, 0, 202},
			want:    202,
			wantErr: nil,
		},
		{
			name:    "large byte slice",
			bytes:   []byte{0, 0, 0, 128, 0, 0, 0, 0},
			want:    128,
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := bytesInt32ToInt64(tt.bytes)
			if err != tt.wantErr {
				t.Errorf("bytesInt32ToInt64() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("bytesInt32ToInt64() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_bytesInt64ToInt64(t *testing.T) {
	tests := []struct {
		name    string
		bytes   []byte
		want    int64
		wantErr error
	}{
		{
			name:    "too short byte slice",
			bytes:   []byte{0, 0, 0, 0},
			want:    0,
			wantErr: errNotEnoughBytes,
		},
		{
			name:    "exact size byte slice",
			bytes:   []byte{0, 0, 0, 0, 0, 0, 0, 202},
			want:    202,
			wantErr: nil,
		},
		{
			name:    "large byte slice",
			bytes:   []byte{0, 0, 0, 0, 0, 0, 0, 128, 0, 0, 0, 0},
			want:    128,
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := bytesInt64ToInt64(tt.bytes)
			if err != tt.wantErr {
				t.Errorf("bytesInt64ToInt64() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("bytesInt64ToInt64() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_bytesFloat64ToFloat64(t *testing.T) {
	tests := []struct {
		name    string
		bytes   []byte
		want    float64
		wantErr error
	}{
		{
			name:    "too short byte slice",
			bytes:   []byte{0, 0, 0, 0},
			want:    0,
			wantErr: errNotEnoughBytes,
		},
		{
			name:    "exact size byte slice",
			bytes:   []byte{64, 9, 33, 251, 84, 68, 45, 24},
			want:    3.141592653589793,
			wantErr: nil,
		},
		{
			name:    "large byte slice",
			bytes:   []byte{64, 9, 33, 251, 84, 68, 45, 24, 0, 0, 0, 0},
			want:    3.141592653589793,
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := bytesFloat64ToFloat64(tt.bytes)
			if err != tt.wantErr {
				t.Errorf("bytesFloat64ToFloat64() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("bytesFloat64ToFloat64() = %v, want %v", got, tt.want)
			}
		})
	}
}
