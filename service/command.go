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

package service // import "go.opentelemetry.io/collector/service"

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/converter/overwritepropertiesconverter"
	"go.opentelemetry.io/collector/service/featuregate"
)

func printComponentSet(out io.Writer, factories component.Factories) {
	stabilityString := func(
		traces component.StabilityLevel,
		metrics component.StabilityLevel,
		logs component.StabilityLevel,
	) string {
		var stability []string

		if traces != component.StabilityLevelUndefined {
			stability = append(stability, fmt.Sprintf("traces: %s", traces))
		}

		if metrics != component.StabilityLevelUndefined {
			stability = append(stability, fmt.Sprintf("metrics: %s", metrics))
		}

		if logs != component.StabilityLevelUndefined {
			stability = append(stability, fmt.Sprintf("logs: %s", logs))
		}

		return strings.Join(stability, ", ")
	}

	fmt.Fprintf(out, "exporters:\n")
	for _, e := range factories.Exporters {
		fmt.Fprintf(out, "\t%s (%s)\n",
			e.Type(),
			stabilityString(e.TracesExporterStability(), e.MetricsExporterStability(), e.LogsExporterStability()),
		)
	}

	fmt.Fprintf(out, "receivers:\n")
	for _, r := range factories.Receivers {
		fmt.Fprintf(out, "\t%s (%s)\n",
			r.Type(),
			stabilityString(r.TracesReceiverStability(), r.MetricsReceiverStability(), r.LogsReceiverStability()),
		)
	}

	fmt.Fprintf(out, "processors:\n")
	for _, p := range factories.Processors {
		fmt.Fprintf(out, "\t%s (%s)\n",
			p.Type(),
			stabilityString(p.TracesProcessorStability(), p.MetricsProcessorStability(), p.LogsProcessorStability()),
		)
	}

	fmt.Fprintf(out, "extensions:\n")
	for _, e := range factories.Extensions {
		fmt.Fprintf(out, "\t%s (%s)\n", e.Type(), e.ExtensionStability())
	}
}

// NewCommand constructs a new cobra.Command using the given CollectorSettings.
func NewCommand(set CollectorSettings) *cobra.Command {
	flagSet := flags()
	rootCmd := &cobra.Command{
		Use:          set.BuildInfo.Command,
		Version:      set.BuildInfo.Version,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if getListComponentsFlag(flagSet) {
				printComponentSet(cmd.OutOrStdout(), set.Factories)
				return nil
			}
			if err := featuregate.GetRegistry().Apply(gatesList); err != nil {
				return err
			}
			if set.ConfigProvider == nil {
				var err error

				configFlags := getConfigFlag(flagSet)
				if len(configFlags) == 0 {
					return errors.New("at least one config flag must be provided")
				}

				cfgSet := newDefaultConfigProviderSettings(configFlags)
				// Append the "overwrite properties converter" as the first converter.
				cfgSet.ResolverSettings.Converters = append(
					[]confmap.Converter{overwritepropertiesconverter.New(getSetFlag(flagSet))},
					cfgSet.ResolverSettings.Converters...)
				set.ConfigProvider, err = NewConfigProvider(cfgSet)
				if err != nil {
					return err
				}
			}
			col, err := New(set)
			if err != nil {
				return err
			}
			return col.Run(cmd.Context())
		},
	}

	rootCmd.Flags().AddGoFlagSet(flagSet)
	return rootCmd
}
