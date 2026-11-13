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
		CorePrefix + "receiver/otlpreceiver":                {"v0.161.0", "v0.162.0"},
		CorePrefix + "processor/batchprocessor":             {"v0.162.0"},
		CorePrefix + "exporter/debugexporter":               {"v0.162.0"},
		CorePrefix + "extension/zpagesextension":            {"v0.161.0"},
		ContribPrefix + "extension/storage/filestorage":     {"v0.162.0"},
		ContribPrefix + "receiver/simpleprometheusreceiver": {"v0.162.0"},
		CorePrefix + "connector/forwardconnector":           {"v1.67.0", "v1.68.0"},
		ContribPrefix + "processor/k8sattributesprocessor":  {"v0.162.0"},
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
		{Component: Component{Class: Receiver, Type: "otlp"}, Module: CorePrefix + "receiver/otlpreceiver", Version: "v0.162.0"},
		{Component: Component{Class: Receiver, Type: "prometheus_simple"}, Module: ContribPrefix + "receiver/simpleprometheusreceiver", Version: "v0.162.0"},
		{Component: Component{Class: Processor, Type: "batch"}, Module: CorePrefix + "processor/batchprocessor", Version: "v0.162.0"},
		{Component: Component{Class: Processor, Type: "k8sattributes"}, Module: ContribPrefix + "processor/k8sattributesprocessor", Version: "v0.162.0"},
		{Component: Component{Class: Exporter, Type: "debug"}, Module: CorePrefix + "exporter/debugexporter", Version: "v0.162.0"},
		{Component: Component{Class: Extension, Type: "file_storage"}, Module: ContribPrefix + "extension/storage/filestorage", Version: "v0.162.0"},
		{Component: Component{Class: Connector, Type: "forward"}, Module: CorePrefix + "connector/forwardconnector", Version: "v1.68.0"},
	}, got.Resolved)

	assert.Equal(t, "v0.162.0", got.Release)
	assert.Equal(t, []Unresolved{
		{Component: Component{Class: Extension, Type: "myvendor"}},
		{Component: Component{Class: Extension, Type: "zpages"}, NotPublished: true},
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
	require.ErrorContains(t, err, "needs access to the Go module proxy")

	// The module proxy is unreachable: nothing can be resolved.
	r.Versions = fixedVersions
	_, err = r.Resolve(t.Context(), []byte("receivers:\n  otlp:\n"))
	require.ErrorIs(t, err, errLookupUnavailable)
}
