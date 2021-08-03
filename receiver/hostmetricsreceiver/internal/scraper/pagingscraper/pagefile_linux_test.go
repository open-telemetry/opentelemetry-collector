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

// +build linux

package pagingscraper

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const validFile = `Filename				Type		Size		Used		Priority
/dev/dm-2                               partition	67022844	490788		-2
/swapfile                               file		2		1		-3
`

const invalidFile = `INVALID				Type		Size		Used		Priority
/dev/dm-2                               partition	67022844	490788		-2
/swapfile                               file		1048572		0		-3
`

func TestGetPageFileStats_ValidFile(t *testing.T) {
	assert := assert.New(t)
	stats, err := getPageFileStatsFromFile(strings.NewReader(validFile))
	assert.NoError(err)

	assert.Equal(*stats[0], pageFileStats{
		deviceName: "/dev/dm-2",
		usedBytes:  502566912,
		freeBytes:  68128825344,
	})

	assert.Equal(*stats[1], pageFileStats{
		deviceName: "/swapfile",
		usedBytes:  1024,
		freeBytes:  1024,
	})
}

func TestGetPageFileStats_InvalidFile(t *testing.T) {
	_, err := getPageFileStatsFromFile(strings.NewReader(invalidFile))
	assert.Error(t, err)
}

func TestGetPageFileStats_EmptyFile(t *testing.T) {
	_, err := getPageFileStatsFromFile(strings.NewReader(""))
	assert.Error(t, err)
}
