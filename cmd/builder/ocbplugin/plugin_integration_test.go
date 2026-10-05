package ocbplugin

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildDummyPlugin(t *testing.T) string {
	t.Helper()

	binName := "dummyplugin"
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	binPath := filepath.Join(t.TempDir(), binName)
	// This is testing code so the nosec is fine here.
	//nolint:gosec // #nosec G204
	cmd := exec.Command("go", "build", "-o", binPath, "../testdata/dummyplugin")
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "failed to build dummyplugin: %s", string(out))
	return binPath
}

func runDummyPluginSubprocess(t *testing.T, binPath string, args ...string) (string, string, error) {
	t.Helper()

	var stdout, stderr bytes.Buffer
	// This is testing code so the nosec is fine here.
	//nolint:gosec // #nosec G204
	cmd := exec.Command(binPath, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// Run the cmd/builder/testdata/dummyplugin under various circumstances
// to test the actual RunPlugin CLI contract. All test cases should be added
// here to ensure we only build the plugin once to save on CI times.
func TestRunPlugin_Subprocess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping subprocess integration test in short mode")
	}

	binPath := buildDummyPlugin(t)

	tests := []struct {
		name         string
		input        string
		noArgs       bool
		noInputFile  bool
		wantStdout   string
		wantStderr   string
		wantExitCode int
	}{
		{
			name:       "pre-generate",
			input:      "action: pre-generate\nocb_version: v0.151.0\nconfig:\n  message: foo\n",
			wantStdout: "pre-generate:foo",
		},
		{
			name:       "post-generate",
			input:      "action: post-generate\nocb_version: v0.151.0\nconfig:\n  message: foo\n",
			wantStdout: "post-generate:foo",
		},
		{
			name:       "pre-build",
			input:      "action: pre-build\nocb_version: v0.151.0\nconfig:\n  message: foo\n",
			wantStdout: "pre-build:foo",
		},
		{
			name:       "post-build",
			input:      "action: post-build\nocb_version: v0.151.0\nconfig:\n  message: foo\n",
			wantStdout: "post-build:foo",
		},
		{
			name:         "unsupported ocb version exits code 1",
			input:        "action: pre-generate\nocb_version: v0.150.0\nconfig:\n  message: foo\n",
			wantStderr:   ErrUnsupportedOCBVersion.Error(),
			wantExitCode: 1,
		},
		{
			name:         "unsupported hook action exits code 1",
			input:        "action: pre-generate\nocb_version: v0.151.0\nconfig:\n  unsupported: true\n",
			wantStderr:   ErrUnsupportedActionPreGenerate.Error(),
			wantExitCode: 1,
		},
		{
			name:         "action error exits code 1",
			input:        "action: pre-build\nocb_version: v0.151.0\nconfig:\n  error: custom failure\n",
			wantStderr:   "error running 'pre-build' plugin action: custom failure",
			wantExitCode: 1,
		},
		{
			name:         "unknown action exits code 1",
			input:        "action: invalid-action\nocb_version: v0.151.0\n",
			wantStderr:   ErrUnknownAction.Error(),
			wantExitCode: 1,
		},
		{
			name:         "input file exists but empty exits code 1",
			wantStderr:   "the plugin input file at",
			wantExitCode: 1,
		},
		{
			name:         "input file does not exist exits code 1",
			noInputFile:  true,
			wantStderr:   "no such file or directory",
			wantExitCode: 1,
		},
		{
			name:         "missing args prints usage exits code 2",
			noArgs:       true,
			wantStderr:   "usage:",
			wantExitCode: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputFilePath := filepath.Join(t.TempDir(), "input.yaml")
			if !tt.noInputFile {
				require.NoError(t, os.WriteFile(inputFilePath, []byte(tt.input), 0o600))
			}
			args := make([]string, 0, 1)
			if !tt.noArgs {
				args = append(args, inputFilePath)
			}

			stdout, stderr, err := runDummyPluginSubprocess(t, binPath, args...)

			if tt.wantExitCode > 0 {
				var exitErr *exec.ExitError
				require.ErrorAs(t, err, &exitErr)
				require.Equal(t, tt.wantExitCode, exitErr.ExitCode())
			} else {
				require.NoError(t, err)
			}

			assert.Contains(t, stderr, tt.wantStderr)
			assert.Contains(t, stdout, tt.wantStdout)
		})
	}
}
