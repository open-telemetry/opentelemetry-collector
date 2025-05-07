// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

//go:generate mdatagen metadata.yaml

import (
	"github.com/spf13/cobra"

	"go.opentelemetry.io/collector/cmd/mdatagen/internal"
)

func main() {
	cmd, err := internal.NewCommand()
	cobra.CheckErr(err)
	cobra.CheckErr(cmd.Execute())
}
