// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package fromconfig

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func paths(cs []candidate) []string {
	out := make([]string, 0, len(cs))
	for _, c := range cs {
		out = append(out, c.path)
	}
	return out
}

func TestCandidatesSimpleType(t *testing.T) {
	got := candidates(Component{Receiver, "otlp"})
	assert.Equal(t, []candidate{
		{0, corePrefix + "receiver/otlpreceiver"},
		{1, contribPrefix + "receiver/otlpreceiver"},
		{2, corePrefix + "receiver/otlp"},
		{3, contribPrefix + "receiver/otlp"},
	}, got)
}

func TestCandidatesExtensionAuth(t *testing.T) {
	got := paths(candidates(Component{Extension, "oauth2client"}))
	assert.Contains(t, got, contribPrefix+"extension/oauth2clientauthextension")
	assert.NotContains(t, paths(candidates(Component{Receiver, "oauth2client"})), contribPrefix+"receiver/oauth2clientauthreceiver")
}

func TestCandidatesKnownComponents(t *testing.T) {
	for _, tt := range []struct {
		component Component
		want      string
	}{
		{Component{Extension, "file_storage"}, contribPrefix + "extension/storage/filestorage"},
		{Component{Extension, "redis_storage"}, contribPrefix + "extension/storage/redisstorageextension"},
		{Component{Extension, "k8s_observer"}, contribPrefix + "extension/observer/k8sobserver"},
		{Component{Extension, "json_log_encoding"}, contribPrefix + "extension/encoding/jsonlogencodingextension"},
		{Component{Extension, "oauth2client"}, contribPrefix + "extension/oauth2clientauthextension"},
		{Component{Receiver, "prometheus_simple"}, contribPrefix + "receiver/simpleprometheusreceiver"},
		{Component{Receiver, "receiver_creator"}, contribPrefix + "receiver/receivercreator"},
		{Component{Receiver, "hostmetrics"}, contribPrefix + "receiver/hostmetricsreceiver"},
		{Component{Processor, "batch"}, corePrefix + "processor/batchprocessor"},
	} {
		t.Run(string(tt.component.Class)+"/"+tt.component.Type, func(t *testing.T) {
			assert.Contains(t, paths(candidates(tt.component)), tt.want)
		})
	}
}

func TestCandidatesPriority(t *testing.T) {
	got := candidates(Component{Extension, "file_storage"})
	require.NotEmpty(t, got)

	// Levels never decrease, and core is tried before contrib at each pattern.
	for i := 1; i < len(got); i++ {
		assert.GreaterOrEqual(t, got[i].level, got[i-1].level)
	}
	assert.Equal(t, corePrefix+"extension/filestorageextension", got[0].path)
	assert.Equal(t, contribPrefix+"extension/filestorageextension", got[1].path)

	// Pattern #4 precedes pattern #5.
	p := paths(got)
	assert.Less(t,
		indexOf(p, contribPrefix+"extension/storage/filestorageextension"),
		indexOf(p, contribPrefix+"extension/storage/filestorage"))
}

func TestCandidatesDeduplicated(t *testing.T) {
	got := paths(candidates(Component{Receiver, "a_b"}))
	seen := map[string]bool{}
	for _, p := range got {
		assert.False(t, seen[p], "duplicate %s", p)
		seen[p] = true
	}
}

func TestCandidatesExceptions(t *testing.T) {
	for _, tt := range []struct {
		component Component
		want      string
	}{
		{Component{Extension, "asapclient"}, contribPrefix + "extension/asapauthextension"},
		{Component{Receiver, "podman_stats"}, contribPrefix + "receiver/podmanreceiver"},
		{Component{Exporter, "otlp_grpc"}, corePrefix + "exporter/otlpexporter"},
	} {
		t.Run(tt.component.Type, func(t *testing.T) {
			assert.Equal(t, []candidate{{0, tt.want}}, candidates(tt.component))
		})
	}
}

func TestCandidatesInvalidPaths(t *testing.T) {
	// A trailing underscore yields an empty path element, which is not a valid module path.
	for _, got := range candidates(Component{Receiver, "foo_"}) {
		assert.NotContains(t, got.path, "//")
		assert.NotContains(t, got.path, "/receiver/receiver")
	}
	assert.Empty(t, candidates(Component{Receiver, ""}))
	assert.Empty(t, candidates(Component{Receiver, "../evil"}))
}

func indexOf(s []string, v string) int {
	for i := range s {
		if s[i] == v {
			return i
		}
	}
	return -1
}
