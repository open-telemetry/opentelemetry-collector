// Copyright 2020, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package hostmetricsreceiver

import (
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-collector/config"
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/open-telemetry/opentelemetry-collector/receiver/hostmetricsreceiver/internal"
	"github.com/open-telemetry/opentelemetry-collector/receiver/hostmetricsreceiver/internal/scraper/cpuscraper"
	"github.com/open-telemetry/opentelemetry-collector/receiver/hostmetricsreceiver/internal/scraper/diskscraper"
	"github.com/open-telemetry/opentelemetry-collector/receiver/hostmetricsreceiver/internal/scraper/filesystemscraper"
	"github.com/open-telemetry/opentelemetry-collector/receiver/hostmetricsreceiver/internal/scraper/memoryscraper"
	"github.com/open-telemetry/opentelemetry-collector/receiver/hostmetricsreceiver/internal/scraper/networkscraper"
)

func TestLoadConfig(t *testing.T) {
	factories, err := config.ExampleComponents()
	require.NoError(t, err)

	factory := NewFactory()
	factories.Receivers[typeStr] = factory
	cfg, err := config.LoadConfigFile(t, path.Join(".", "testdata", "config.yaml"), factories)

	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, len(cfg.Receivers), 2)

	r0 := cfg.Receivers["hostmetrics"]
	defaultConfigAllScrapers := factory.CreateDefaultConfig()
	defaultConfigAllScrapers.(*Config).Scrapers = map[string]internal.Config{
		cpuscraper.TypeStr: (&cpuscraper.Factory{}).CreateDefaultConfig(),
	}
	assert.Equal(t, r0, defaultConfigAllScrapers)

	r1 := cfg.Receivers["hostmetrics/customname"].(*Config)
	assert.Equal(t, r1,
		&Config{
			ReceiverSettings: configmodels.ReceiverSettings{
				TypeVal: typeStr,
				NameVal: "hostmetrics/customname",
			},
			CollectionInterval: 30 * time.Second,
			Scrapers: map[string]internal.Config{
				cpuscraper.TypeStr:        &cpuscraper.Config{ReportPerCPU: true},
				diskscraper.TypeStr:       &diskscraper.Config{},
				filesystemscraper.TypeStr: &filesystemscraper.Config{},
				memoryscraper.TypeStr:     &memoryscraper.Config{},
				networkscraper.TypeStr:    &networkscraper.Config{},
			},
		})
}
