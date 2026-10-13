// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"os"
	"path/filepath"
	"testing"

	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/pdata/internal/metadata"
)

func BenchmarkAppleByteArena(b *testing.B) {
	dir := os.Getenv("OTEL_CMP_DIR")
	if dir == "" {
		dir = "/tmp/otel-cmp"
	}
	cases := []struct {
		name string
		new  func() interface{ UnmarshalProtoState([]byte, *State) error }
	}{
		{"string-traces", func() interface{ UnmarshalProtoState([]byte, *State) error } {
			return NewExportTraceServiceRequest()
		}},
		{"string-logs", func() interface{ UnmarshalProtoState([]byte, *State) error } {
			return NewExportLogsServiceRequest()
		}},
		{"dense-traces", func() interface{ UnmarshalProtoState([]byte, *State) error } {
			return NewExportTraceServiceRequest()
		}},
		{"dense-logs", func() interface{ UnmarshalProtoState([]byte, *State) error } {
			return NewExportLogsServiceRequest()
		}},
		{"dense-metrics", func() interface{ UnmarshalProtoState([]byte, *State) error } {
			return NewExportMetricsServiceRequest()
		}},
	}
	for _, c := range cases {
		buf, err := os.ReadFile(filepath.Join(dir, c.name+".bin"))
		if err != nil {
			b.Logf("skip %s: %v", c.name, err)
			continue
		}
		b.Run(c.name, func(b *testing.B) {
			for _, pooling := range []bool{false, true} {
				name := "pooling=off"
				if pooling {
					name = "pooling=on"
				}
				b.Run(name, func(b *testing.B) {
					prev := metadata.PdataUseProtoPoolingFeatureGate.IsEnabled()
					if err := featuregate.GlobalRegistry().Set(metadata.PdataUseProtoPoolingFeatureGate.ID(), pooling); err != nil {
						b.Fatal(err)
					}
					b.Cleanup(func() {
						_ = featuregate.GlobalRegistry().Set(metadata.PdataUseProtoPoolingFeatureGate.ID(), prev)
					})
					decode := func() {
						st := NewState()
						st.RetainWire(buf)
						if err := c.new().UnmarshalProtoState(buf, st); err != nil {
							b.Fatal(err)
						}
						st.DropArena()
					}
					decode()
					b.SetBytes(int64(len(buf)))
					b.ReportAllocs()
					for b.Loop() {
						decode()
					}
				})
			}
		})
	}
}
