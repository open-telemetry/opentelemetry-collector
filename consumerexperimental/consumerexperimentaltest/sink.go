// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumerexperimentaltest // import "go.opentelemetry.io/collector/consumer/consumerexperimentaltest"

import (
	"context"
	"sync"

	"go.opentelemetry.io/collector/consumerexperimental"
	"go.opentelemetry.io/collector/pdata/pprofile"
)

// ProfilesSink is a consumer.Profiles that acts like a sink that
// stores all profiles and allows querying them for testing.
type ProfilesSink struct {
	nonMutatingConsumer
	mu            sync.Mutex
	profiles      []pprofile.Profiles
	profilesCount int
}

var _ consumerexperimental.Profiles = (*ProfilesSink)(nil)

// ConsumeProfiles stores profiles to this sink.
func (ste *ProfilesSink) ConsumeProfiles(_ context.Context, td pprofile.Profiles) error {
	ste.mu.Lock()
	defer ste.mu.Unlock()

	ste.profiles = append(ste.profiles, td)
	ste.profilesCount += 1

	return nil
}

// AllProfiles returns the profiles stored by this sink since last Reset.
func (ste *ProfilesSink) AllProfiles() []pprofile.Profiles {
	ste.mu.Lock()
	defer ste.mu.Unlock()

	copyProfiles := make([]pprofile.Profiles, len(ste.profiles))
	copy(copyProfiles, ste.profiles)
	return copyProfiles
}

// SpanCount returns the number of spans sent to this sink.
func (ste *ProfilesSink) ProfilesCount() int {
	ste.mu.Lock()
	defer ste.mu.Unlock()
	return ste.profilesCount
}

// Reset deletes any stored data.
func (ste *ProfilesSink) Reset() {
	ste.mu.Lock()
	defer ste.mu.Unlock()

	ste.profiles = nil
	ste.profilesCount = 0
}
