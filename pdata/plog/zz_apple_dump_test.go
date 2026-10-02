// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package plog

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDumpDenseLogs(t *testing.T) {
	if os.Getenv("OTEL_DUMP_CMP") == "" {
		t.Skip()
	}
	buf := dumpLogsAtLeast(10<<20, func(n int) []byte {
		ld := generateBenchmarkLogsPayload(n)
		out, err := (&ProtoMarshaler{}).MarshalLogs(ld)
		if err != nil {
			t.Fatal(err)
		}
		ld.getState().DropArena()
		return out
	})
	writeLogsDump(t, "dense-logs.bin", buf)
}

func dumpLogsAtLeast(target int, gen func(int) []byte) []byte {
	n := 2_000
	buf := gen(n)
	n = int(float64(n)*float64(target)/float64(len(buf))) + 1
	buf = gen(n)
	for len(buf) < target {
		n = n*target/len(buf) + n/10 + 1
		buf = gen(n)
	}
	return buf
}

func writeLogsDump(t *testing.T, name string, buf []byte) {
	t.Helper()
	dir := os.Getenv("OTEL_CMP_DIR")
	if dir == "" {
		dir = "/tmp/otel-cmp"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, buf, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %s bytes=%d", path, len(buf))
}
