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

package testbed

import (
	"os"
	"strconv"
	"testing"

	"go.opentelemetry.io/collector/testutil"
)

func GetAvailablePort(t *testing.T) int {
	return int(testutil.GetAvailablePort(t))
}

func GetScrapeInterval() int {
	scrapeInterval, err := strconv.Atoi(os.Getenv("SCRAPE_INTERVAL"))
	if err != nil {
		return 1
	} else {
		return scrapeInterval
	}
}

func GetItemsPerInterval() int {
	itemsPerInterval, err := strconv.Atoi(os.Getenv("ITEMS_PER_INTERVAL"))
	if err != nil {
		return 10_000
	} else {
		return itemsPerInterval
	}
}
