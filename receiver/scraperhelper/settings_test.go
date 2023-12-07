// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scraperhelper

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestScrapeControllerSettings(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		set    ScraperControllerSettings
		errVal string
	}{
		{
			name:   "default configuration",
			set:    NewDefaultScraperControllerSettings(""),
			errVal: "",
		},
		{
			name:   "zero value configuration",
			set:    ScraperControllerSettings{},
			errVal: `"collection_interval": requires positive value`,
		},
		{
			name: "invalid timeout",
			set: ScraperControllerSettings{
				CollectionInterval: time.Minute,
				Timeout:            -1 * time.Minute,
			},
			errVal: `"timeout": requires positive value`,
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.set.Validate()
			if tc.errVal == "" {
				assert.NoError(t, err, "Must not error")
				return
			}
			assert.EqualError(t, err, tc.errVal, "Must match the expected error")
		})
	}
}
