// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package service // import "go.opentelemetry.io/collector/service"

import (
	"fmt"
	"os"
	"text/tabwriter"

	"go.opentelemetry.io/collector/service/internal/graph"
)

func DisplayFeatures() error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "ID\tEnabled\tStage\tDescription\n")
	data := graph.GetFeaturesTableData()
	for _, row := range data.Rows {
		fmt.Fprintf(w, "%s\t%v\t%s\t%s\n",
			row.ID,
			row.Enabled,
			row.Stage,
			row.Description)
	}
	return w.Flush()
}

func DisplayFeature(id string) error {
	data := graph.GetFeaturesTableData()
	for _, row := range data.Rows {
		if row.ID == id {
			fmt.Printf("Feature: %s\n", row.ID)
			fmt.Printf("Enabled: %v\n", row.Enabled)
			fmt.Printf("Stage: %s\n", row.Stage)
			fmt.Printf("Description: %s\n", row.Description)
			fmt.Printf("From Version: %s\n", row.FromVersion)
			if row.ToVersion != "" {
				fmt.Printf("To Version: %s\n", row.ToVersion)
			}
			return nil
		}
	}
	return fmt.Errorf("feature %q not found", id)
}
