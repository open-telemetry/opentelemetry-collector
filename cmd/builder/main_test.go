// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"text/template"
	"time"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/cmd/builder/internal"
	"go.opentelemetry.io/collector/cmd/builder/internal/builder"
)

func renderTmplFile(t *testing.T, filePath string, outPath string, data map[string]any) {
	tmplPath := filepath.Join("testdata", filePath)
	tplBytes, err := os.ReadFile(tmplPath) //nolint:gosec
	require.NoError(t, err, "read template: %s", tmplPath)
	tpl, err := template.New(filepath.Base(tmplPath)).Parse(string(tplBytes))
	require.NoError(t, err, "parse template: %s", tmplPath)
	var buf bytes.Buffer
	err = tpl.Execute(&buf, data)
	require.NoError(t, err, "exec template: %s", tmplPath)
	err = os.WriteFile(outPath, buf.Bytes(), 0o600)
	require.NoError(t, err, "write %s", outPath)
}

func TestCollectorBuildAndRun(t *testing.T) {
	tests := []struct {
		builderYAML   string
		runConfigYAML string
		healthPort    int
	}{
		{
			builderYAML:   "core.builder.yaml",
			runConfigYAML: "core.otel.yaml",
			healthPort:    55680,
		},
		{
			builderYAML:   "package.builder.yaml",
			runConfigYAML: "package.otel.yaml",
			healthPort:    55699,
		},
	}
	for _, tt := range tests {
		t.Run(tt.builderYAML, func(t *testing.T) {
			tmpdir := t.TempDir()

			collectorBin := getCollectorBin(t, tmpdir, tt.builderYAML)

			var colBuf bytes.Buffer
			runConfig, err := filepath.Abs(filepath.Join("testdata", tt.runConfigYAML))
			require.NoError(t, err)

			otelCmd := exec.Command(collectorBin, "--config", runConfig) //nolint:gosec
			otelCmd.Stdout = &colBuf
			otelCmd.Stderr = &colBuf
			otelCmd.Dir = filepath.Dir(collectorBin)
			require.NoError(t, otelCmd.Start())
			defer func() {
				_ = otelCmd.Process.Kill()
				_, _ = otelCmd.Process.Wait()
			}()

			servicez := fmt.Sprintf("http://localhost:%d/debug/servicez", tt.healthPort)
			ok := false
			deadline := time.Now().Add(20 * time.Second)
			for time.Now().Before(deadline) {
				resp, err := http.Get(servicez) //nolint:gosec
				if err == nil && resp.StatusCode == http.StatusOK {
					_, err = io.Copy(io.Discard, resp.Body)
					require.NoError(t, err)
					err = resp.Body.Close()
					require.NoError(t, err)
					ok = true
					break
				}
				time.Sleep(200 * time.Millisecond)
			}
			if !ok {
				t.Fatalf("collector did not become healthy. Collector output:\n%s", colBuf.String())
			}
		})
	}
}

func getCollectorBin(t *testing.T, tmpdir string, yamlPath string) string {
	renderedYAML := filepath.Join(tmpdir, "builder.rendered.yaml")

	renderTmplFile(t,
		yamlPath,
		renderedYAML,
		map[string]any{"OutputPath": tmpdir},
	)

	cfg := unmarshalConf(t, renderedYAML)

	packageName := cfg.Distribution.Package

	binName := cfg.Distribution.Name
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}

	collectorBin := filepath.Join(tmpdir, binName)
	buildCollector(t, renderedYAML)

	if packageName == "" || packageName == "main" {
		require.FileExists(t, collectorBin)
		return collectorBin
	}

	// build wrapper
	builderPkgDir := cfg.Distribution.OutputPath

	renderTmplFile(t,
		"export.go.tql",
		filepath.Join(builderPkgDir, "export.go"),
		map[string]any{"Package": packageName},
	)

	wrapperDir := filepath.Join(tmpdir, "wrapper")
	require.NoError(t, os.MkdirAll(wrapperDir, 0o700))
	renderTmplFile(t,
		"main.go.tql",
		filepath.Join(wrapperDir, "main.go"),
		map[string]any{"Package": packageName, "ImportPath": packageName},
	)
	renderTmplFile(t,
		"go.mod.tql",
		filepath.Join(wrapperDir, "go.mod"),
		map[string]any{
			"Package":    packageName,
			"ImportPath": packageName,
			"PkgDir":     builderPkgDir,
		},
	)

	require.NoError(t, runCmd(wrapperDir, "go", "mod", "tidy"))

	collectorBin = filepath.Join(wrapperDir, binName)
	require.NoError(t, runCmd(wrapperDir, "go", "build", "-o", collectorBin, "main.go"))

	require.FileExists(t, collectorBin)
	return collectorBin
}

func runCmd(dir string, name string, arg ...string) error {
	cmd := exec.Command(name, arg...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s execute failed:%s", name, string(out))
	}
	return nil
}

func buildCollector(t *testing.T, configPath string) {
	cmd, err := internal.Command()
	require.NoError(t, err)
	cmd.SetArgs([]string{"--config", configPath})
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetErr(&outBuf)
	err = cmd.Execute()
	require.NoError(t, err, "builder execution failed: %s", outBuf.String())
}

func unmarshalConf(t *testing.T, yamlPath string) builder.Config {
	k := koanf.New(".")
	require.NoError(t, k.Load(file.Provider(yamlPath), yaml.Parser()))

	cfg := builder.Config{}
	require.NoError(t, k.UnmarshalWithConf("", &cfg, koanf.UnmarshalConf{Tag: "mapstructure"}))

	return cfg
}
