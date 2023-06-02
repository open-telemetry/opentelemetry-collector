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

package processscraper

import (
	"context"
	"reflect"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal"
	"go.opentelemetry.io/collector/receiver/scraperhelper"
	"go.uber.org/zap"
)

func TestCreateDefaultConfig(t *testing.T) {
	factory := &Factory{}
	cfg := factory.CreateDefaultConfig()
	assert.IsType(t, &Config{}, cfg)
}

func TestCreateResourceMetricsScraper(t *testing.T) {
	factory := &Factory{}
	cfg := &Config{}

	scraper, err := factory.CreateResourceMetricsScraper(context.Background(), zap.NewNop(), cfg)

	if runtime.GOOS == "linux" || runtime.GOOS == "windows" {
		assert.NoError(t, err)
		assert.NotNil(t, scraper)
	} else {
		assert.Error(t, err)
		assert.Nil(t, scraper)
	}
}

func TestFactory_CreateDefaultConfig(t *testing.T) {
	tests := []struct {
		name string
		f    *Factory
		want internal.Config
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.f.CreateDefaultConfig(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Factory.CreateDefaultConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFactory_CreateResourceMetricsScraper(t *testing.T) {
	type args struct {
		in0    context.Context
		in1    *zap.Logger
		config internal.Config
	}
	tests := []struct {
		name    string
		f       *Factory
		args    args
		want    scraperhelper.ResourceMetricsScraper
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.f.CreateResourceMetricsScraper(tt.args.in0, tt.args.in1, tt.args.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Factory.CreateResourceMetricsScraper() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Factory.CreateResourceMetricsScraper() = %v, want %v", got, tt.want)
			}
		})
	}
}
