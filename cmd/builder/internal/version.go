// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/cmd/builder/internal"

import (
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"
)

var version = ""

func init() {
	// the second returned value is a boolean, which is true if the binaries are built with module support.
	if version != "" {
		return
	}
	info, ok := debug.ReadBuildInfo()
	if ok {
		version = info.Main.Version
	}
}

func versionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Version of ocb",
		Long:  "Prints the version of the ocb binary",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Println(fmt.Sprintf("%s version %s", cmd.Parent().Name(), version))
			return nil
		},
	}
}
