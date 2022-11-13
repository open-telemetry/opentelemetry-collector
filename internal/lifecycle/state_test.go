// Copyright  The OpenTelemetry Authors
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

package lifecycle

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLifecycleState(t *testing.T) {
	t.Parallel()

	state := NewState()
	assert.True(t, state.Start(), "Must confirm the state has been updated")
	assert.False(t, state.Start(), "Must confirm the state no state change")

	assert.True(t, state.Stop(), "Must confirm the state has been updated")
	assert.False(t, state.Stop(), "Must confirm the state no state change")
}

func TestLifecycleConcurrency(t *testing.T) {
	t.Parallel()

	state := NewState()

	var (
		updated int
		start   = make(chan struct{})
		wg      sync.WaitGroup
	)

	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if state.Start() {
				updated++
			}
		}()
	}
	close(start)
	wg.Wait()
	assert.Equal(t, 1, updated, "Must have only been updated once")

	start = make(chan struct{})

	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if state.Stop() {
				updated--
			}
		}()
	}
	close(start)
	wg.Wait()
	assert.Equal(t, 0, updated, "Must have only been updated once")
}
