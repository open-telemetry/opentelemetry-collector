// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/cmd/builder/internal"

import (
	"bufio"
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/cmd/builder/internal/builder"
)

func TestInitCommand(t *testing.T) {
	cmd := initCommand()

	assert.NotNil(t, cmd)
	assert.IsType(t, &cobra.Command{}, cmd)
	assert.Equal(t, "init", cmd.Use)
}

func TestRunInit(t *testing.T) {
	for _, tt := range []struct {
		name      string
		buildPath func(string) string

		wantErr string
	}{
		{
			name:      "without a path",
			buildPath: func(string) string { return "" },
			wantErr:   "argument must be a folder",
		},
		{
			name:      "with a relative path",
			buildPath: func(string) string { return "./tmp/init" },
			wantErr:   "",
		},
		{
			name:      "with an absolute path",
			buildPath: func(dir string) string { return dir },
			wantErr:   "",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tmpdir := filepath.Join(t.TempDir(), "init")
			path := tt.buildPath(tmpdir)
			err := run(path)

			if tt.wantErr == "" {
				require.NoError(t, err)
				validateCollector(t, path)
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

func TestInitRunCollector(t *testing.T) {
	path := filepath.Join(t.TempDir(), "run")

	err := run(path)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "make", "run")
	cmd.Dir = path
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to get stdout pipe: %v", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatalf("Failed to get stderr pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start process: %v", err)
	}

	found := make(chan bool, 1)

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			t.Logf("STDOUT: %s", line)
			if strings.Contains(line, "Everything is ready.") {
				found <- true
				return
			}
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			t.Logf("STDERR: %s", line)
			if strings.Contains(line, "Everything is ready.") {
				found <- true
				return
			}
		}
	}()

	assert.Eventually(t, func() bool {
		select {
		case <-found:
			return true
		default:
			return false
		}
	}, 5*time.Minute, 100*time.Millisecond, "Collector should start up and print 'Everything is ready.'")

	t.Log("Collector started up correctly - killing process")
	cancel()

	require.NoError(t, cmd.Wait())
}

func validateCollector(t *testing.T, path string) {
	require.FileExists(t, filepath.Join(path, ".gitignore"))
	require.FileExists(t, filepath.Join(path, "manifest.yaml"))
	require.FileExists(t, filepath.Join(path, "go.mod"))
	require.FileExists(t, filepath.Join(path, "go.sum"))
	require.FileExists(t, filepath.Join(path, "Makefile"))
	require.FileExists(t, filepath.Join(path, "config.yaml"))

	k := koanf.New(".")
	err := k.Load(file.Provider(filepath.Join(path, "manifest.yaml")), yaml.Parser())
	require.NoError(t, err)

	cfg := builder.Config{}
	err = k.UnmarshalWithConf("", &cfg, koanf.UnmarshalConf{
		Tag: "mapstructure",
	})
	require.NoError(t, err)

	assert.Equal(t, "init", cfg.Distribution.Name)
	assert.Equal(t, defaultDescription, cfg.Distribution.Description)
}
