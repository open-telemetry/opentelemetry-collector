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

package scrapererror

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPartialScrapeError(t *testing.T) {
	failed := 2
	err := fmt.Errorf("some error")
	partialErr := NewPartialScrapeError(err, failed)
	assert.Equal(t, err.Error(), partialErr.Error())
	assert.Equal(t, failed, partialErr.Failed)
}

func TestIsPartialScrapeError(t *testing.T) {
	err := errors.New("testError")
	require.False(t, IsPartialScrapeError(err))

	err = NewPartialScrapeError(err, 2)
	require.True(t, IsPartialScrapeError(err))
}
