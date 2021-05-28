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

package hostmetricsreceiver

import (
	"errors"
	"fmt"

	"github.com/spf13/cast"

	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configparser"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal"
	"go.opentelemetry.io/collector/receiver/scraperhelper"
)

const (
	scrapersKey = "scrapers"
)

// Config defines configuration for HostMetrics receiver.
type Config struct {
	scraperhelper.ScraperControllerSettings `mapstructure:",squash"`
	Scrapers                                map[string]internal.Config `mapstructure:"-"`
}

var _ config.Receiver = (*Config)(nil)
var _ config.CustomUnmarshable = (*Config)(nil)

// Validate checks the receiver configuration is valid
func (cfg *Config) Validate() error {
	if len(cfg.Scrapers) == 0 {
		return errors.New("must specify at least one scraper when using hostmetrics receiver")
	}

	return nil
}

// Unmarshal a config.Parser into the config struct.
func (cfg *Config) Unmarshal(componentParser *configparser.Parser) error {
	if componentParser == nil {
		return nil
	}

	// load the non-dynamic config normally
	err := componentParser.Unmarshal(cfg)
	if err != nil {
		return err
	}

	// dynamically load the individual collector configs based on the key name

	cfg.Scrapers = map[string]internal.Config{}

	scrapersSection, err := componentParser.Sub(scrapersKey)
	if err != nil {
		return err
	}

	for key := range cast.ToStringMap(componentParser.Get(scrapersKey)) {
		factory, ok := getScraperFactory(key)
		if !ok {
			return fmt.Errorf("invalid scraper key: %s", key)
		}

		collectorCfg := factory.CreateDefaultConfig()
		collectorViperSection, err := scrapersSection.Sub(key)
		if err != nil {
			return err
		}
		err = collectorViperSection.UnmarshalExact(collectorCfg)
		if err != nil {
			return fmt.Errorf("error reading settings for scraper type %q: %v", key, err)
		}

		cfg.Scrapers[key] = collectorCfg
	}

	return nil
}
