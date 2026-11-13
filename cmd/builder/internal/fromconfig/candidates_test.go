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
	got := candidates(Component{Class: Receiver, Type: "otlp"})
	assert.Equal(t, []candidate{
		{0, corePrefix + "receiver/otlpreceiver"},
		{1, contribPrefix + "receiver/otlpreceiver"},
		{2, corePrefix + "receiver/otlp"},
		{3, contribPrefix + "receiver/otlp"},
	}, got)
}

func TestCandidatesExtensionAuth(t *testing.T) {
	got := paths(candidates(Component{Class: Extension, Type: "oauth2client"}))
	assert.Contains(t, got, contribPrefix+"extension/oauth2clientauthextension")
	assert.NotContains(t, paths(candidates(Component{Class: Receiver, Type: "oauth2client"})), contribPrefix+"receiver/oauth2clientauthreceiver")
}

func TestCandidatesKnownComponents(t *testing.T) {
	for _, tt := range []struct {
		component Component
		want      string
	}{
		{Component{Class: Extension, Type: "file_storage"}, contribPrefix + "extension/storage/filestorage"},
		{Component{Class: Extension, Type: "redis_storage"}, contribPrefix + "extension/storage/redisstorageextension"},
		{Component{Class: Extension, Type: "k8s_observer"}, contribPrefix + "extension/observer/k8sobserver"},
		{Component{Class: Extension, Type: "json_log_encoding"}, contribPrefix + "extension/encoding/jsonlogencodingextension"},
		{Component{Class: Extension, Type: "oauth2client"}, contribPrefix + "extension/oauth2clientauthextension"},
		{Component{Class: Receiver, Type: "prometheus_simple"}, contribPrefix + "receiver/simpleprometheusreceiver"},
		{Component{Class: Receiver, Type: "receiver_creator"}, contribPrefix + "receiver/receivercreator"},
		{Component{Class: Receiver, Type: "hostmetrics"}, contribPrefix + "receiver/hostmetricsreceiver"},
		{Component{Class: Processor, Type: "batch"}, corePrefix + "processor/batchprocessor"},
	} {
		t.Run(string(tt.component.Class)+"/"+tt.component.Type, func(t *testing.T) {
			assert.Contains(t, paths(candidates(tt.component)), tt.want)
		})
	}
}

func TestCandidatesPriority(t *testing.T) {
	got := candidates(Component{Class: Extension, Type: "file_storage"})
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
	got := paths(candidates(Component{Class: Receiver, Type: "a_b"}))
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
		{Component{Class: Extension, Type: "asapclient"}, contribPrefix + "extension/asapauthextension"},
		{Component{Class: Receiver, Type: "podman_stats"}, contribPrefix + "receiver/podmanreceiver"},
		{Component{Class: Exporter, Type: "otlp_grpc"}, corePrefix + "exporter/otlpexporter"},
	} {
		t.Run(tt.component.Type, func(t *testing.T) {
			assert.Equal(t, []candidate{{0, tt.want}}, candidates(tt.component))
		})
	}
}

func TestCandidatesInvalidPaths(t *testing.T) {
	// A trailing underscore yields an empty path element, which is not a valid module path.
	for _, got := range candidates(Component{Class: Receiver, Type: "foo_"}) {
		assert.NotContains(t, got.path, "//")
		assert.NotContains(t, got.path, "/receiver/receiver")
	}
	assert.Empty(t, candidates(Component{Class: Receiver, Type: ""}))
	assert.Empty(t, candidates(Component{Class: Receiver, Type: "../evil"}))
}

func indexOf(s []string, v string) int {
	for i := range s {
		if s[i] == v {
			return i
		}
	}
	return -1
}
