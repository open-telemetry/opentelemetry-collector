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
		{"internal", Receiver, CorePrefix + "receiver/internal/foo", true},
		{"examples", Exporter, ContribPrefix + "exporter/examples/foo", true},
		{"class test", Receiver, CorePrefix + "receiver/receivertest", true},
		{"class helper", Exporter, CorePrefix + "exporter/exporterhelper", true},
		{"x class", Receiver, CorePrefix + "receiver/xreceiver", true},
		{"x class helper", Exporter, CorePrefix + "exporter/xexporterhelper", true},
		{"other class helper", Exporter, CorePrefix + "exporter/receiverhelper", true},
		{"ends with test", Receiver, ContribPrefix + "receiver/integrationtest", true},
		{"ends with helper", Processor, ContribPrefix + "processor/scraperhelper", true},
		{"x prefix and class suffix", Receiver, ContribPrefix + "receiver/xfoo" + "receiver", true},
		{"test in the middle", Receiver, ContribPrefix + "receiver/latencytestreceiver", false},
		{"x prefix without class suffix", Receiver, ContribPrefix + "receiver/xenon", false},
		{"denylist storage", Extension, ContribPrefix + "extension/storage", true},
		{"denylist observer", Extension, ContribPrefix + "extension/observer", true},
		{"denylist encoding", Extension, ContribPrefix + "extension/encoding", true},
		{"denylist opampcustommessages", Extension, ContribPrefix + "extension/opampcustommessages", true},
		{"denylist extensionauth", Extension, CorePrefix + "extension/extensionauth", true},
		{"denylist extensioncapabilities", Extension, CorePrefix + "extension/extensioncapabilities", true},
		{"denylist extensionmiddleware", Extension, CorePrefix + "extension/extensionmiddleware", true},
		{"denylist is exact, not a prefix", Extension, ContribPrefix + "extension/storage/filestorage", false},
		{"nested observer", Extension, ContribPrefix + "extension/observer/k8sobserver", false},
		{"regular component", Receiver, CorePrefix + "receiver/otlpreceiver", false},
		{"regular contrib component", Receiver, ContribPrefix + "receiver/hostmetricsreceiver", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isNotComponent(tt.class, tt.path))
		})
	}
}

func TestCandidatesSkipNotComponents(t *testing.T) {
	// "storage" would otherwise resolve to the extension/storage module through pattern #2.
	got := paths(candidates(Component{Class: Extension, Type: "storage"}))
	assert.NotContains(t, got, ContribPrefix+"extension/storage")
	assert.Contains(t, got, ContribPrefix+"extension/storageextension")

	got = paths(candidates(Component{Class: Extension, Type: "extensionauth"}))
	assert.NotContains(t, got, CorePrefix+"extension/extensionauth")
}
