package internal

import "github.com/spf13/cobra"

func createCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "[EXPERIMENTAL] Creates a custom collector repository based on the provided configuration",
		Long:  `ocb create creates a custom collector repository in the provided folder with a manifest to start building a custom collector component. This command is experimental and very likely to change.`,
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runCreate()
		},
	}

	cmd.AddCommand(initExporterCommand())

	return cmd
}

func runCreate() error {
	return nil
}
