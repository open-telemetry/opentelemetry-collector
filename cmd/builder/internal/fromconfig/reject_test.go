// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package fromconfig

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsNotComponent(t *testing.T) {
	for _, tt := range []struct {
		name  string
		class Class
		path  string
		want  bool
	}{
		{"internal", Receiver, corePrefix + "receiver/internal/foo", true},
		{"examples", Exporter, contribPrefix + "exporter/examples/foo", true},
		{"test suffix", Receiver, contribPrefix + "receiver/foointegrationtest", true},
		{"helper suffix", Exporter, corePrefix + "exporter/exporterhelper", true},
		{"x class", Receiver, corePrefix + "receiver/xreceiver", true},
		{"x class helper", Exporter, corePrefix + "exporter/xexporterhelper", true},
		{"x prefix class suffix", Processor, corePrefix + "processor/xprocessor", true},
		{"denylist storage", Extension, contribPrefix + "extension/storage", true},
		{"denylist observer", Extension, contribPrefix + "extension/observer", true},
		{"denylist encoding", Extension, contribPrefix + "extension/encoding", true},
		{"denylist opampcustommessages", Extension, contribPrefix + "extension/opampcustommessages", true},
		{"denylist extensionauth", Extension, corePrefix + "extension/extensionauth", true},
		{"denylist extensioncapabilities", Extension, corePrefix + "extension/extensioncapabilities", true},
		{"denylist extensionmiddleware", Extension, corePrefix + "extension/extensionmiddleware", true},
		{"denylist is exact, not a prefix", Extension, contribPrefix + "extension/storage/filestorage", false},
		{"nested observer", Extension, contribPrefix + "extension/observer/k8sobserver", false},
		{"regular component", Receiver, corePrefix + "receiver/otlpreceiver", false},
		{"regular contrib component", Receiver, contribPrefix + "receiver/hostmetricsreceiver", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isNotComponent(tt.class, tt.path))
		})
	}
}

func TestCandidatesSkipNotComponents(t *testing.T) {
	// "storage" would otherwise resolve to the extension/storage module through pattern #2.
	got := paths(candidates(Component{Extension, "storage"}))
	assert.NotContains(t, got, contribPrefix+"extension/storage")
	assert.Contains(t, got, contribPrefix+"extension/storageextension")

	got = paths(candidates(Component{Extension, "extensionauth"}))
	assert.NotContains(t, got, corePrefix+"extension/extensionauth")
}
