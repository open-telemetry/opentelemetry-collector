package internal

import "github.com/spf13/cobra"

func initExporterCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "exporter",
		Short: "Initializes a new custom exporter repository in the provided folder",
		Long:  `ocb create exporter initializes a new repository in the provided folder with a manifest to start building a custom Collector exporter. This command is experimental and very likely to change.`,
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runExporter()
		},
	}
}

func runExporter() error {
	return nil
}
