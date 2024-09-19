// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"runtime/debug"
	"testing"

	"github.com/spf13/cobra"
)

// Mock debug.ReadBuildInfo function
var readBuildInfo = debug.ReadBuildInfo

func TestBinVersion(t *testing.T) {
	// Test case: version is set
	version = "v1.0.0"
	v, err := binVersion(readBuildInfo)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if v != "v1.0.0" {
		t.Fatalf("expected version 'v1.0.0', got %v", v)
	}

	// // Test case: version is not set, ReadBuildInfo returns valid info
	version = ""
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Main: debug.Module{
				Version: "v2.0.0",
			},
		}, true
	}
	v, err = binVersion(readBuildInfo)
	fmt.Printf("v: %v, err: %v", v, err)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if v != "v2.0.0" {
		t.Fatalf("expected version 'v2.0.0', got %v", v)
	}

	// Test case: version is not set, ReadBuildInfo fails
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return nil, false
	}
	v, err = binVersion(readBuildInfo)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if v != "" {
		t.Fatalf("expected empty version, got %v", v)
	}
}

var validBinVersionFunc binVersionFunc = func(_ debugReadBuildInfoFunc) (string, error) {
	return "v1.0.0", nil
}

var invalidBinVersionFunc binVersionFunc = func(_ debugReadBuildInfoFunc) (string, error) {
	return "", fmt.Errorf("failed to get version")
}

func TestVersionCommand(t *testing.T) {
	tests := []struct {
		name           string
		binVersion     binVersionFunc
		expectedOutput string
		expectedError  bool
	}{
		{
			name:           "valid version",
			binVersion:     validBinVersionFunc,
			expectedOutput: "ocb version v1.0.0\n",
			expectedError:  false,
		},
		{
			name:           "error in binVersion",
			binVersion:     invalidBinVersionFunc,
			expectedOutput: "",
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set a mock parent command name
			parentCmd := &cobra.Command{
				Use: "ocb <command>",
			}
			// Create the command
			var cmd = versionCommand(tt.binVersion)
			parentCmd.AddCommand(cmd)
			// Capture the output
			output := bytes.NewBufferString("")
			errOutput := bytes.NewBufferString("")
			cmd.SetOut(output)
			cmd.SetErr(errOutput)
			// Create a new context with a fake value
			type contextKey string
			ctx := context.WithValue(context.Background(), contextKey("key"), "value")
			// Set fake CLI arguments
			fakeArgs := []string{"cmd", "version"}
			os.Args = fakeArgs
			// Execute the command
			err := cmd.ExecuteContext(ctx)
			// Check for expected error
			if (err != nil) != tt.expectedError {
				t.Fatalf("expected error: %v, got: %v", tt.expectedError, err)
			}
			// Check for expected output
			if output.String() != tt.expectedOutput {
				t.Fatalf("expected output: %v, got: %v", tt.expectedOutput, output.String())
			}
		})
	}
}
