// Copyright 2019 OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sampling

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsSampled(t *testing.T) {
	DIVISOR := 100
	// Do Not Sample
	var sampler Sampler = NewCountingSampler(0.0)
	assert.Equal(t, getSampleBucketSize(sampler, DIVISOR), 0)

	// Sample 10%
	sampler = NewCountingSampler(0.1)
	assert.Equal(t, getSampleBucketSize(sampler, DIVISOR), 10)

	// Sample 20%
	sampler = NewCountingSampler(0.2)
	assert.Equal(t, getSampleBucketSize(sampler, DIVISOR), 20)

	// Sample 50%
	sampler = NewCountingSampler(0.5)
	assert.Equal(t, getSampleBucketSize(sampler, DIVISOR), 50)

	// Sample 100%
	sampler = NewCountingSampler(1.0)
	assert.Equal(t, getSampleBucketSize(sampler, DIVISOR), 100)
}

func getSampleBucketSize(sampler Sampler, traceCount int) int {
	j := 0
	for i := 0; i < traceCount; i++ {
		if sampler.isSampled() {
			j++
		}
	}
	return j
}
