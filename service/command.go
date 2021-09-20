// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package service

import (
	"flag"

	"github.com/spf13/cobra"

	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/internal/collector/telemetry"
	"go.opentelemetry.io/collector/service/internal/telemetrylogs"
	"go.opentelemetry.io/collector/service/parserprovider"
)

// NewCommand constructs a new cobra.Command using the given Collector.
// TODO: Make this independent of the collector internals.
func NewCommand(col *Collector) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:          col.set.BuildInfo.Command,
		Version:      col.set.BuildInfo.Version,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return col.Run(cmd.Context())
		},
	}

	// TODO: coalesce this code and expose this information to other components.
	flagSet := new(flag.FlagSet)
	addFlagsFns := []func(*flag.FlagSet){
		configtelemetry.Flags,
		parserprovider.Flags,
		telemetry.Flags,
		telemetrylogs.Flags,
	}
	for _, addFlags := range addFlagsFns {
		addFlags(flagSet)
	}

	rootCmd.Flags().AddGoFlagSet(flagSet)
	return rootCmd
}
