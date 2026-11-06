// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package fromconfig

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fixedVersions(context.Context, string) (Versions, error) {
	return Versions{Beta: "v0.162.0", Stable: "v1.68.0"}, nil
}

func TestResolve(t *testing.T) {
	proxy := &fakeProxy{modules: map[string][]string{
		corePrefix + "receiver/otlpreceiver":                {"v0.161.0", "v0.162.0"},
		corePrefix + "processor/batchprocessor":             {"v0.162.0"},
		corePrefix + "exporter/debugexporter":               {"v0.162.0"},
		corePrefix + "extension/zpagesextension":            {"v0.161.0"},
		contribPrefix + "extension/storage/filestorage":     {"v0.162.0"},
		contribPrefix + "receiver/simpleprometheusreceiver": {"v0.162.0"},
		corePrefix + "connector/forwardconnector":           {"v1.67.0", "v1.68.0"},
		contribPrefix + "processor/k8sattributesprocessor":  {"v0.162.0"},
	}}
	r := &Resolver{Release: "v0.162.0", List: proxy.list, Versions: fixedVersions}

	got, err := r.Resolve(t.Context(), []byte(`
receivers:
  otlp:
  otlp/2:
  prometheus_simple:
processors:
  batch:
  k8sattributes:
exporters:
  debug:
extensions:
  file_storage:
  zpages:
  myvendor:
connectors:
  forward:
`))
	require.NoError(t, err)

	assert.Equal(t, []Resolved{
		{Component{Receiver, "otlp"}, corePrefix + "receiver/otlpreceiver", "v0.162.0"},
		{Component{Receiver, "prometheus_simple"}, contribPrefix + "receiver/simpleprometheusreceiver", "v0.162.0"},
		{Component{Processor, "batch"}, corePrefix + "processor/batchprocessor", "v0.162.0"},
		{Component{Processor, "k8sattributes"}, contribPrefix + "processor/k8sattributesprocessor", "v0.162.0"},
		{Component{Exporter, "debug"}, corePrefix + "exporter/debugexporter", "v0.162.0"},
		{Component{Extension, "file_storage"}, contribPrefix + "extension/storage/filestorage", "v0.162.0"},
		{Component{Connector, "forward"}, corePrefix + "connector/forwardconnector", "v1.68.0"},
	}, got.Resolved)

	assert.Equal(t, []Unresolved{
		{Component{Extension, "myvendor"}, `no Go module found for extension "myvendor"`},
		{Component{Extension, "zpages"}, `the Go module of extension "zpages" is not published at release v0.162.0`},
	}, got.Unresolved)
}

func TestResolveGoMod(t *testing.T) {
	r := Resolved{Module: "example.com/m", Version: "v1.0.0"}
	assert.Equal(t, "example.com/m v1.0.0", r.GoMod())
}

func TestResolveNoComponents(t *testing.T) {
	proxy := &fakeProxy{}
	r := &Resolver{Release: "v0.162.0", List: proxy.list, Versions: func(context.Context, string) (Versions, error) {
		t.Fatal("versions must not be looked up without components")
		return Versions{}, nil
	}}
	got, err := r.Resolve(t.Context(), []byte("service: {}"))
	require.NoError(t, err)
	assert.Empty(t, got.Resolved)
	assert.Empty(t, got.Unresolved)
	assert.Empty(t, proxy.calls)
}

func TestResolveErrors(t *testing.T) {
	proxy := &fakeProxy{}

	r := &Resolver{Release: "v0.162.0", List: proxy.list, Versions: fixedVersions}
	_, err := r.Resolve(t.Context(), []byte("receivers: ${env:FOO}"))
	require.ErrorContains(t, err, "must be a mapping")

	r.Versions = func(context.Context, string) (Versions, error) { return Versions{}, assert.AnError }
	_, err = r.Resolve(t.Context(), []byte("receivers:\n  otlp:\n"))
	require.ErrorIs(t, err, assert.AnError)

	// The module proxy is unreachable: nothing can be resolved.
	r.Versions = fixedVersions
	_, err = r.Resolve(t.Context(), []byte("receivers:\n  otlp:\n"))
	require.ErrorIs(t, err, errLookupUnavailable)
}
