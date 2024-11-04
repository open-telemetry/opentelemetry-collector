// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

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
			if (err != nil) != tt.wantErr {
				t.Errorf("Command() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
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
		})
	}
}

func Test_applyCfgFromFile(t *testing.T) {
	testDistribution := builder.Distribution{
		Module:           "testModule",
		Name:             "testName",
		Go:               "testGO",
		Description:      "testDescription",
		OtelColVersion:   "testOtelColVersion",
		OutputPath:       "testOutputPath",
		Version:          "testVersion",
		BuildTags:        "",
		DebugCompilation: true,
	}
	testStringTable := []string{"A", "B", "C"}
	testModule := builder.Module{
		Name:   "testName",
		GoMod:  "testGoMod",
		Import: "testImport",
		Path:   "testPath",
	}
	type args struct {
		cfgFromFile builder.Config
	}
	tests := []struct {
		name    string
		args    args
		want    builder.Config
		wantErr bool
	}{
		{
			name: "distribution, scheme, excludes, exporters, receivers, processors, replaces are applied correctly",
			args: args{
				cfgFromFile: builder.Config{
					Logger:       zap.NewNop(),
					Distribution: testDistribution,
					Excludes:     testStringTable,
					Processors:   []builder.Module{testModule},
					Receivers:    []builder.Module{testModule},
					Exporters:    []builder.Module{testModule},
					Replaces:     testStringTable,
					ConfResolver: builder.ConfResolver{
						DefaultURIScheme: "env",
					},
				},
			},
			want: builder.Config{
				Logger:       zap.NewNop(),
				Distribution: testDistribution,
				ConfResolver: builder.ConfResolver{
					DefaultURIScheme: "env",
				},
				Excludes:   testStringTable,
				Processors: []builder.Module{testModule},
				Receivers:  []builder.Module{testModule},
				Exporters:  []builder.Module{testModule},
				Replaces:   testStringTable,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flags := flag.NewFlagSet("version=1.0.0", 1)

			applyCfgFromFile(flags, tt.args.cfgFromFile)
			assert.Equal(t, tt.want.ConfResolver.DefaultURIScheme, cfg.ConfResolver.DefaultURIScheme)
			assert.Equal(t, tt.want.Distribution, cfg.Distribution)
			assert.Equal(t, tt.want.Excludes, cfg.Excludes)
			assert.Equal(t, tt.want.Exporters, cfg.Exporters)
			assert.Equal(t, tt.want.Receivers, cfg.Receivers)
			assert.Equal(t, tt.want.Processors, cfg.Processors)
			assert.Equal(t, tt.want.Replaces, cfg.Replaces)
		})
	}
}

func Test_initConfig(t *testing.T) {
	type args struct {
		flags *flag.FlagSet
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "initConfig created correctly",
			args: args{
				flags: flag.NewFlagSet("version=1.0.0", 1),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := initConfig(tt.args.flags); (err != nil) != tt.wantErr {
				t.Errorf("initConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
