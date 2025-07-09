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

	"go.opentelemetry.io/collector/cmd/builder/internal"

	"github.com/stretchr/testify/require"
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
		packageName   string
	}{
		// {
		// 	builderYAML:   "core.otel.yaml",
		// 	runConfigYAML: "otel.yaml",
		// 	packageName:   "main",
		// },
		{
			builderYAML:   "package.otel.yaml",
			runConfigYAML: "otel.yaml",
			packageName:   "collector",
		},
	}
	for _, tt := range tests {
		t.Run(tt.builderYAML, func(t *testing.T) {
			tmpdir := t.TempDir()
			// todo should be read config, not use packageName
			collectorBin := getCollectorBin(t, tmpdir, tt.builderYAML, tt.packageName)

			var colBuf bytes.Buffer
			runConfig := filepath.Join("testdata", tt.runConfigYAML)
			otelCmd := exec.Command(collectorBin, "--config", runConfig) //nolint:gosec
			otelCmd.Stdout = &colBuf
			otelCmd.Stderr = &colBuf
			otelCmd.Dir = filepath.Dir(collectorBin)
			require.NoError(t, otelCmd.Start())
			defer func() {
				_ = otelCmd.Process.Kill()
				_, _ = otelCmd.Process.Wait()
			}()
			// todo test is parallel
			servicez := "http://localhost:55679/debug/servicez"
			ok := false
			deadline := time.Now().Add(20 * time.Second)
			for time.Now().Before(deadline) {
				resp, err := http.Get(servicez)
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

func getCollectorBin(t *testing.T, tmpdir string, yamlPath string, packageName string) string {
	if packageName == "" || packageName == "main" {
		collectorBin := filepath.Join(tmpdir, "otelcol-test")
		renderedYAML := filepath.Join(tmpdir, "builder.rendered.yaml")
		renderTmplFile(t,
			yamlPath,
			renderedYAML,
			map[string]any{"OutputPath": tmpdir},
		)
		buildCollector(t, renderedYAML)
		require.FileExists(t, collectorBin)
		return collectorBin
	}
	// build wrapper
	builderPkgDir := filepath.Join(tmpdir, packageName)
	require.NoError(t, os.MkdirAll(builderPkgDir, 0o700))
	renderedYAML := filepath.Join(tmpdir, "builder.rendered.yaml")
	renderTmplFile(t,
		yamlPath,
		renderedYAML,
		map[string]any{"OutputPath": builderPkgDir},
	)
	buildCollector(t, renderedYAML)

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

	binName := "otelcol-test"
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}

	collectorBin := filepath.Join(wrapperDir, binName)
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
