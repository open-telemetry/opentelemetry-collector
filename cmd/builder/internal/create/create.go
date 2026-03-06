// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package create // import "go.opentelemetry.io/collector/cmd/builder/internal/create"

import "github.com/spf13/cobra"

func CreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "[EXPERIMENTAL] Creates a custom collector repository based on the provided configuration",
		Long:  `ocb create creates a custom collector repository in the provided folder with a manifest to start building a custom collector component. This command is experimental and very likely to change.`,
	}

	cmd.AddCommand(newExporterCommand())

	return cmd
}
