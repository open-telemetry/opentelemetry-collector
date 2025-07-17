// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confmap_test

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"go.opentelemetry.io/collector/confmap"
)

type DiskScrape struct {
	Disk   string        `mapstructure:"disk"`
	Scrape time.Duration `mapstructure:"scrape"`
}

// We can annotate a struct with mapstructure field annotations.
func Example_simpleUnmarshaling() {
	conf := confmap.NewFromStringMap(map[string]any{
		"disk":   "c",
		"scrape": "5s",
	})
	scrapeInfo := &DiskScrape{}
	if err := conf.Unmarshal(scrapeInfo); err != nil {
		panic(err)
	}
	fmt.Printf("Configuration contains the following:\nDisk: %q\nScrape: %s\n", scrapeInfo.Disk, scrapeInfo.Scrape)
	// Output: Configuration contains the following:
	// Disk: "c"
	// Scrape: 5s
}

type CPUScrape struct {
	Enabled bool `mapstructure:"enabled"`
}

type ComputerScrape struct {
	DiskScrape `mapstructure:",squash"`
	CPUScrape  `mapstructure:",squash"`
}

// We can unmarshal embedded structs with mapstructure field annotations.
func Example_embeddedUnmarshaling() {
	conf := confmap.NewFromStringMap(map[string]any{
		"disk":    "c",
		"scrape":  "5s",
		"enabled": true,
	})
	scrapeInfo := &ComputerScrape{}
	if err := conf.Unmarshal(scrapeInfo); err != nil {
		panic(err)
	}
	fmt.Printf("Configuration contains the following:\nDisk: %q\nScrape: %s\nEnabled: %v\n", scrapeInfo.Disk, scrapeInfo.Scrape, scrapeInfo.Enabled)
	// Output: Configuration contains the following:
	// Disk: "c"
	// Scrape: 5s
	// Enabled: true
}

type NetworkScrape struct {
	Enabled  bool     `mapstructure:"enabled"`
	Networks []string `mapstructure:"networks"`
	Wifi     bool     `mapstructure:"wifi"`
}

func (n *NetworkScrape) Unmarshal(c *confmap.Conf) error {
	if err := c.Unmarshal(n, confmap.WithIgnoreUnused()); err != nil {
		return err
	}
	if slices.Contains(n.Networks, "wlan0") {
		n.Wifi = true
	}
	return nil
}

type ManualScrapeInfo struct {
	Disk   string
	Scrape time.Duration
}

func (m *ManualScrapeInfo) Unmarshal(c *confmap.Conf) error {
	m.Disk = c.Get("disk").(string)
	if c.Get("vinyl") == "33" {
		m.Scrape = 10 * time.Second
	} else {
		m.Scrape = 2 * time.Second
	}
	return nil
}

type RouterScrape struct {
	NetworkScrape `mapstructure:",squash"`
}

// We can unmarshal an embedded struct with a custom `Unmarshal` method.
func Example_embeddedManualUnmarshaling() {
	conf := confmap.NewFromStringMap(map[string]any{
		"networks": []string{"eth0", "eth1", "wlan0"},
		"enabled":  true,
	})
	scrapeInfo := &RouterScrape{}
	if err := conf.Unmarshal(scrapeInfo); err != nil {
		panic(err)
	}
	fmt.Printf("Configuration contains the following:\nNetworks: %q\nWifi: %v\nEnabled: %v\n", scrapeInfo.Networks, scrapeInfo.Wifi, scrapeInfo.Enabled)
	// Output: Configuration contains the following:
	// Networks: ["eth0" "eth1" "wlan0"]
	// Wifi: true
	// Enabled: true
}

func Example_manualUnmarshaling() {
	conf := confmap.NewFromStringMap(map[string]any{
		"disk":  "Beatles",
		"vinyl": "33",
	})
	scrapeInfo := &ManualScrapeInfo{}
	if err := conf.Unmarshal(scrapeInfo, confmap.WithIgnoreUnused()); err != nil {
		panic(err)
	}
	fmt.Printf("Configuration contains the following:\nDisk: %q\nScrape: %s\n", scrapeInfo.Disk, scrapeInfo.Scrape)
	// Output: Configuration contains the following:
	// Disk: "Beatles"
	// Scrape: 10s
}

// mockFileProvider simulates the standard file provider behavior.
// In typical usage, it reads a single file as the root configuration.
// This mock implementation always returns a fixed configuration:
// { "my-config": "${expand:to-expand}" }
type mockFileProvider struct{}

func (d mockFileProvider) Retrieve(ctx context.Context, uri string, watcher confmap.WatcherFunc) (*confmap.Retrieved, error) {
	expectedUri := "file:mock-file"
	if uri != expectedUri {
		panic("should not happen, the uri is expected to be " + expectedUri + " for mockFileProvider")
	}
	return confmap.NewRetrieved(map[string]any{
		"my-config": "${expand:to-expand}",
	})
}

func (d mockFileProvider) Scheme() string {
	return "file"
}

func (d mockFileProvider) Shutdown(ctx context.Context) error {
	return nil
}

// mockExpandProvider simulates a typical inline expansion provider.
// In configurations, you can use expressions like ${SCHEMA:VALUE},
// where the provider associated with SCHEMA is responsible for resolving the value.
type mockExpandProvider struct{}

func (m mockExpandProvider) Retrieve(ctx context.Context, uri string, watcher confmap.WatcherFunc) (*confmap.Retrieved, error) {
	expectedUri := "expand:to-expand"
	if uri != expectedUri {
		panic("should not happen, the uri is expected to be " + expectedUri + " for mockExpandProvider")
	}
	return confmap.NewRetrieved("expanded")
}

func (m mockExpandProvider) Scheme() string {
	return "expand"
}

func (m mockExpandProvider) Shutdown(ctx context.Context) error {
	return nil
}

// mockUpperCaseConverter transforms the value of the `my-config` field in the configuration to uppercase.
type mockUpperCaseConverter struct{}

func (m mockUpperCaseConverter) Convert(ctx context.Context, conf *confmap.Conf) error {
	currentValue := conf.Get("my-config")
	expectedValue := "expanded"
	if currentValue != expectedValue {
		panic("should not happen, the value for converter should always be " + expectedValue + " for mockUpperCaseConverter")
	}
	upperCaseConf := confmap.NewFromStringMap(map[string]any{
		"my-config": strings.ToUpper(currentValue.(string)),
	})
	if conf.Merge(upperCaseConf) != nil {
		panic("merge failed, this should not happen in this example.")
	}
	return nil
}

func Example_converterAndProvider() {
	resolver, err := confmap.NewResolver(confmap.ResolverSettings{
		URIs: []string{"file:mock-file"},
		ProviderFactories: []confmap.ProviderFactory{
			confmap.NewProviderFactory(func(ps confmap.ProviderSettings) confmap.Provider {
				return &mockFileProvider{}
			}),
			confmap.NewProviderFactory(func(ps confmap.ProviderSettings) confmap.Provider {
				return &mockExpandProvider{}
			}),
		},
		ConverterFactories: []confmap.ConverterFactory{
			confmap.NewConverterFactory(func(settings confmap.ConverterSettings) confmap.Converter {
				return &mockUpperCaseConverter{}
			}),
		},
	})
	if err != nil {
		panic(err)
	}

	conf, err := resolver.Resolve(context.Background())
	if err != nil {
		panic(err)
	}
	fmt.Printf("Configuration contains the following:\nmy-config: %s", conf.Get("my-config"))
	// Output: Configuration contains the following:
	// my-config: EXPANDED
}
