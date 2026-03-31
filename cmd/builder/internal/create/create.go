// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package create // import "go.opentelemetry.io/collector/cmd/builder/internal/create"

import "github.com/spf13/cobra"

func CreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "[EXPERIMENTAL] Creates a new OpenTelemetry Collector component",
		Long:  `ocb create creates a new OpenTelemetry Collector component (exporter, receiver, processor, extension) with the necessary boilerplate files. The generated component can be used in any custom collector built with ocb. This command is experimental and very likely to change.`,
	}

	cmd.AddCommand(newExporterCommand())

	return cmd
}
