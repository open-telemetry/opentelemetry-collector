// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/cmd/builder/internal/builder"
)

func TestCommand(t *testing.T) {
	tests := []struct {
		name    string
		want    *cobra.Command
		wantErr bool
	}{
		{
			name: "command created",
			want: &cobra.Command{
				SilenceUsage:  true, // Don't print usage on Run error.
				SilenceErrors: true, // Don't print errors; main does it.
				Use:           "ocb",
				Long:          "OpenTelemetry Collector Builder",
				Args:          cobra.NoArgs,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Command()
			if !tt.wantErr {
				require.NoErrorf(t, err, "Command()")
				assert.Equal(t, tt.want.Aliases, got.Aliases)
				assert.Equal(t, tt.want.Annotations, got.Annotations)
				assert.Equal(t, tt.want.ValidArgs, got.ValidArgs)
				assert.Equal(t, tt.want.ArgAliases, got.ArgAliases)
				assert.Equal(t, tt.want.Use, got.Use)
				assert.Equal(t, tt.want.SilenceUsage, got.SilenceUsage)
				assert.Equal(t, tt.want.SilenceErrors, got.SilenceErrors)
				assert.True(t, strings.HasPrefix(got.Long, tt.want.Long))
				assert.Empty(t, got.Short)
				assert.NotEqual(t, tt.want.HasFlags(), got.Flags().HasFlags())
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestApplyFlags(t *testing.T) {
	tests := []struct {
		name  string
		flags []string
		want  *builder.Config
	}{
		{
			name: "Default flag values",
			want: &builder.Config{
				SkipStrictVersioning: true,
			},
		},
		{
			name:  "All flag values",
			flags: []string{"--skip-generate=true", "--skip-compilation=true", "--skip-get-modules=true", "--skip-strict-versioning=true", "--ldflags=test", "--gcflags=test", "--verbose=true"},
			want: &builder.Config{
				SkipGenerate:         true,
				SkipCompilation:      true,
				SkipGetModules:       true,
				SkipStrictVersioning: true,
				LDFlags:              "test",
				GCFlags:              "test",
				Verbose:              true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flags := flag.NewFlagSet("version=1.0.0", 1)
			require.NoError(t, initFlags(flags))
			require.NoError(t, flags.Parse(tt.flags))
			cfg, err := builder.NewDefaultConfig()
			require.NoError(t, err)
			require.NoError(t, applyFlags(flags, cfg))
			assert.Equal(t, tt.want.SkipGenerate, cfg.SkipGenerate)
			assert.Equal(t, tt.want.SkipCompilation, cfg.SkipCompilation)
			assert.Equal(t, tt.want.SkipGetModules, cfg.SkipGetModules)
			assert.Equal(t, tt.want.SkipStrictVersioning, cfg.SkipStrictVersioning)
			assert.Equal(t, tt.want.LDFlags, cfg.LDFlags)
			assert.Equal(t, tt.want.Verbose, cfg.Verbose)
		})
	}
}

func TestInitConfig(t *testing.T) {
	tests := []struct {
		name    string
		flags   *flag.FlagSet
		wantErr bool
	}{
		{
			name:    "initConfig created correctly",
			flags:   flag.NewFlagSet("version=1.0.0", 1),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, initFlags(tt.flags))
			_, err := initConfig(tt.flags)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
