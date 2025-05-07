// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"github.com/spf13/cobra"

	"go.opentelemetry.io/collector/cmd/builder/internal"
)

func main() {
	cmd, err := internal.Command()
	cobra.CheckErr(err)
	cobra.CheckErr(cmd.Execute())
}
