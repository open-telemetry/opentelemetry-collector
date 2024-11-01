// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confmap_test

import (
	"fmt"
	"slices"
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
