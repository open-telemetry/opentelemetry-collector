// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package create // import "go.opentelemetry.io/collector/cmd/builder/internal/create"

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunCreateExporter(t *testing.T) {
	for _, tt := range []struct {
		name      string
		buildPath func(string) string
		signals   []string
		wantErr   string
	}{
		{
			name:      "with relative path",
			buildPath: func(string) string { return "./tmp/myexporter" },
			signals:   []string{"traces", "metrics", "logs"},
			wantErr:   "",
		},
		{
			name:      "with absolute path",
			buildPath: func(dir string) string { return dir },
			signals:   []string{"traces", "metrics", "logs"},
			wantErr:   "",
		},
		{
			name:      "with only traces signal",
			buildPath: func(dir string) string { return dir },
			signals:   []string{"traces"},
			wantErr:   "",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tmpdir := filepath.Join(t.TempDir(), "myexporter")
			path := tt.buildPath(tmpdir)
			err := runCreateExporter("my", tt.signals, path)

			if tt.wantErr == "" {
				require.NoError(t, err)
				validateExporter(t, filepath.Join(path, "myexporter"))
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}
