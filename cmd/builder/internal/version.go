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
)

type debugReadBuildInfoFunc func() (info *debug.BuildInfo, ok bool)
type binVersionFunc func(fn debugReadBuildInfoFunc) (string, error)

// binVersion returns the version of the binary.
// If the version is not set, it attempts to read the build information.
// Returns an error if the build information cannot be read.
func binVersion(fn debugReadBuildInfoFunc) (string, error) {
	if version != "" {
		return version, nil
	}
	info, ok := fn()
	if !ok {
		return "", fmt.Errorf("failed to read build info")
	}
	return info.Main.Version, nil
}

func versionCommand(fn binVersionFunc) *cobra.Command {
	var err error
	return &cobra.Command{
		Use:   "version",
		Short: "Version of ocb",
		Long:  "Prints the version of the ocb binary",
		RunE: func(cmd *cobra.Command, _ []string) error {
			version, err = fn(debug.ReadBuildInfo)
			if err != nil {
				return err
			}
			cmd.Println(fmt.Sprintf("%s version %s", cmd.Parent().Name(), version))
			return nil
		},
	}
}
