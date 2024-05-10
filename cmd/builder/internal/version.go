// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/cmd/builder/internal"

import (
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"
)

var (
	version = ""
	date    = "unknown"
)

// binVersion returns the version of the binary.
// If the version is not set, it attempts to read the build information.
// Returns an error if the build information cannot be read.
func binVersion() (string, error) {
	if version != "" {
		return version, nil
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "", fmt.Errorf("failed to read build info")
	}
	return info.Main.Version, nil
}

func versionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Version of ocb",
		Long:  "Prints the version of the ocb binary",
		RunE: func(cmd *cobra.Command, _ []string) error {
			version, err := binVersion()
			if err != nil {
				return err
			}
			cmd.Println(fmt.Sprintf("%s version %s", cmd.Parent().Name(), version))
			return nil
		},
	}
}
