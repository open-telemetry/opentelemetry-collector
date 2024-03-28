// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confmap_test

import (
	"fmt"
	"time"

	"go.opentelemetry.io/collector/confmap"
)

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
	//Output: Configuration contains the following:
	// Disk: "Beatles"
	// Scrape: 10s
}
