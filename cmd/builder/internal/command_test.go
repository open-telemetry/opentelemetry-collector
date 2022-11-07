// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
		flags       *flag.FlagSet
		cfgFromFile builder.Config
	}
	tests := []struct {
		name    string
		args    args
		want    builder.Config
		wantErr bool
	}{
		{
			name: "distribution, excludes, exporters, receivers, processors, replaces are applied correctly",
			args: args{
				flags: flag.NewFlagSet("version=1.0.0", 1),
				cfgFromFile: builder.Config{
					Logger:       zap.NewNop(),
					Distribution: testDistribution,
					Excludes:     testStringTable,
					Processors:   []builder.Module{testModule},
					Receivers:    []builder.Module{testModule},
					Exporters:    []builder.Module{testModule},
					Replaces:     testStringTable,
				},
			},
			want: builder.Config{
				Logger:       zap.NewNop(),
				Distribution: testDistribution,
				Excludes:     testStringTable,
				Processors:   []builder.Module{testModule},
				Receivers:    []builder.Module{testModule},
				Exporters:    []builder.Module{testModule},
				Replaces:     testStringTable,
			},
			wantErr: false,
		},
		{
			name: "Skip compilation false",
			args: args{
				flags: flag.NewFlagSet("version=1.0.0", 1),
				cfgFromFile: builder.Config{
					Logger:       zap.NewNop(),
					Distribution: testDistribution,
				},
			},
			want: builder.Config{
				Logger:          zap.NewNop(),
				SkipCompilation: false,
				Distribution:    testDistribution,
			},
			wantErr: false,
		},
		{
			name: "Skip compilation true",
			args: args{
				flags: flag.NewFlagSet("version=1.0.0", 1),
				cfgFromFile: builder.Config{
					Logger:          zap.NewNop(),
					SkipCompilation: true,
					Distribution:    testDistribution,
				},
			},
			want: builder.Config{
				Logger:          zap.NewNop(),
				SkipCompilation: true,
				Distribution:    testDistribution,
			},
			wantErr: false,
		},
		{
			name: "Skip get modules false",
			args: args{
				flags: flag.NewFlagSet("version=1.0.0", 1),
				cfgFromFile: builder.Config{
					Logger:          zap.NewNop(),
					SkipCompilation: true,
					Distribution:    testDistribution,
				},
			},
			want: builder.Config{
				Logger:          zap.NewNop(),
				SkipCompilation: true,
				SkipGetModules:  false,
				Distribution:    testDistribution,
			},
			wantErr: false,
		},
		{
			name: "Skip get modules true",
			args: args{
				flags: flag.NewFlagSet("version=1.0.0", 1),
				cfgFromFile: builder.Config{
					Logger:          zap.NewNop(),
					SkipCompilation: true,
					SkipGetModules:  true,
					Distribution:    testDistribution,
				},
			},
			want: builder.Config{
				Logger:          zap.NewNop(),
				SkipCompilation: true,
				SkipGetModules:  true,
				Distribution:    testDistribution,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			applyCfgFromFile(tt.args.flags, tt.args.cfgFromFile)
			assert.Equal(t, tt.want.Distribution, cfg.Distribution)
			assert.Equal(t, tt.want.SkipCompilation, cfg.SkipCompilation)
			assert.Equal(t, tt.want.SkipGetModules, cfg.SkipGetModules)
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
