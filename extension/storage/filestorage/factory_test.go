// Copyright The OpenTelemetry Authors
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

package filestorage

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
)

func TestFactory(t *testing.T) {
	f := NewFactory()
	require.Equal(t, typeStr, f.Type())

	cfg := f.CreateDefaultConfig().(*Config)
	require.Equal(t, config.NewID(typeStr), cfg.ID())

	if runtime.GOOS != "windows" {
		require.Equal(t, "/var/lib/otelcol/file_storage", cfg.Directory)
	} else {
		expected := filepath.Join(os.Getenv("ProgramData"), "Otelcol", "FileStorage")
		require.Equal(t, expected, cfg.Directory)
	}
	require.Equal(t, time.Second, cfg.Timeout)

	tests := []struct {
		name           string
		config         *Config
		wantErr        bool
		wantErrMessage string
	}{
		{
			name:           "Default",
			config:         cfg,
			wantErr:        true,
			wantErrMessage: "directory must exist",
		},
		{
			name: "Invalid directory",
			config: &Config{
				Directory: "/not/very/likely/a/real/dir",
			},
			wantErr:        true,
			wantErrMessage: "directory must exist",
		},
		{
			name: "Default",
			config: func() *Config {
				tempDir, _ := ioutil.TempDir("", "")
				return &Config{
					Directory: tempDir,
				}
			}(),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			e, err := f.CreateExtension(
				context.Background(),
				component.ExtensionCreateParams{
					Logger: zap.NewNop(),
				},
				test.config,
			)
			if test.wantErr {
				if test.wantErrMessage != "" {
					require.True(t, strings.HasPrefix(err.Error(), test.wantErrMessage))
				}
				require.Error(t, err)
				require.Nil(t, e)
			} else {
				require.NoError(t, err)
				require.NotNil(t, e)
				ctx := context.Background()
				require.NoError(t, e.Start(ctx, componenttest.NewNopHost()))
				require.NoError(t, e.Shutdown(ctx))
			}
		})
	}
}
