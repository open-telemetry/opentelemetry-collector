// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package fromconfig

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeProxy is a ListFunc serving a fixed set of modules, recording its calls.
type fakeProxy struct {
	modules map[string][]string
	calls   [][]string
}

func (f *fakeProxy) list(_ context.Context, modules []string) ([]ListedModule, error) {
	f.calls = append(f.calls, modules)
	out := make([]ListedModule, 0, len(modules))
	for _, m := range modules {
		out = append(out, ListedModule{Path: m, Versions: f.modules[m]})
	}
	return out, nil
}

// listEach is a ListFunc answering every module with the result of answer.
func listEach(answer func(path string) ListedModule) ListFunc {
	return func(_ context.Context, modules []string) ([]ListedModule, error) {
		out := make([]ListedModule, 0, len(modules))
		for _, m := range modules {
			out = append(out, answer(m))
		}
		return out, nil
	}
}

func acceptAll(ListedModule) (string, bool) { return "", true }

func TestVerifyHighestPriorityWins(t *testing.T) {
	proxy := &fakeProxy{modules: map[string][]string{
		corePrefix + "receiver/otlpreceiver":                     {"v0.1.0"},
		contribPrefix + "receiver/hostmetricsreceiver":           {"v0.1.0"},
		contribPrefix + "extension/storage/filestorage":          {"v0.1.0"},
		contribPrefix + "extension/storage/filestorageextension": {"v0.1.0"},
	}}
	components := []Component{
		{Receiver, "otlp"},
		{Receiver, "hostmetrics"},
		{Extension, "file_storage"},
	}

	got, err := verify(t.Context(), proxy.list, components, acceptAll)
	require.NoError(t, err)

	assert.Equal(t, corePrefix+"receiver/otlpreceiver", got.modules[components[0]].module.Path)
	assert.Equal(t, contribPrefix+"receiver/hostmetricsreceiver", got.modules[components[1]].module.Path)
	// Pattern #4 outranks pattern #5.
	assert.Equal(t, contribPrefix+"extension/storage/filestorageextension", got.modules[components[2]].module.Path)
}

func TestVerifyBatchesByLevel(t *testing.T) {
	proxy := &fakeProxy{modules: map[string][]string{
		corePrefix + "receiver/otlpreceiver":               {"v0.1.0"},
		contribPrefix + "receiver/hostmetricsreceiver":     {"v0.1.0"},
		corePrefix + "processor/batchprocessor":            {"v0.1.0"},
		contribPrefix + "processor/k8sattributesprocessor": {"v0.1.0"},
		corePrefix + "exporter/debugexporter":              {"v0.1.0"},
		contribPrefix + "exporter/elasticsearchexporter":   {"v0.1.0"},
	}}
	components := []Component{
		{Receiver, "otlp"},
		{Receiver, "hostmetrics"},
		{Processor, "batch"},
		{Processor, "k8sattributes"},
		{Exporter, "debug"},
		{Exporter, "elasticsearch"},
	}

	got, err := verify(t.Context(), proxy.list, components, acceptAll)
	require.NoError(t, err)
	assert.Len(t, got.modules, len(components))

	// One call for core candidates, one for the contrib ones.
	require.Len(t, proxy.calls, 2)
	assert.Contains(t, proxy.calls[0], sentinelModule)
	assert.NotContains(t, proxy.calls[1], sentinelModule)
}

func TestVerifyUnresolved(t *testing.T) {
	proxy := &fakeProxy{modules: map[string][]string{
		sentinelModule: {"v0.1.0"},
	}}
	components := []Component{{Extension, "myvendor"}}

	got, err := verify(t.Context(), proxy.list, components, acceptAll)
	require.NoError(t, err)
	assert.Empty(t, got.modules)
	assert.Empty(t, got.found)
	// All the candidate levels were tried.
	assert.Len(t, proxy.calls, len(candidates(components[0])))
}

func TestVerifyAccept(t *testing.T) {
	proxy := &fakeProxy{modules: map[string][]string{
		sentinelModule:                             {"v0.1.0"},
		corePrefix + "processor/batchprocessor":    {"v0.2.0"},
		contribPrefix + "processor/batchprocessor": {"v0.1.0"},
	}}
	components := []Component{{Processor, "batch"}}

	// The core module is skipped because it is not accepted, the contrib one is used.
	got, err := verify(t.Context(), proxy.list, components, func(m ListedModule) (string, bool) {
		return "v0.1.0", m.Versions[0] == "v0.1.0"
	})
	require.NoError(t, err)
	assert.Equal(t, contribPrefix+"processor/batchprocessor", got.modules[components[0]].module.Path)

	// When nothing is accepted, the component is reported as found.
	got, err = verify(t.Context(), proxy.list, components, func(ListedModule) (string, bool) { return "", false })
	require.NoError(t, err)
	assert.Empty(t, got.modules)
	assert.True(t, got.found[components[0]])
}

func TestVerifySentinelFailure(t *testing.T) {
	for _, tt := range []struct {
		name string
		list ListFunc
		want string
	}{
		{
			name: "error reported",
			list: listEach(func(path string) ListedModule {
				return ListedModule{Path: path, Error: &ListError{Err: "module lookup disabled by GOPROXY=off"}}
			}),
			want: "module lookup disabled by GOPROXY=off",
		},
		{
			name: "no versions",
			list: listEach(func(path string) ListedModule { return ListedModule{Path: path} }),
			want: "module not found",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := verify(t.Context(), tt.list, []Component{{Receiver, "otlp"}}, acceptAll)
			require.ErrorIs(t, err, errLookupUnavailable)
			require.ErrorContains(t, err, tt.want)
			require.ErrorContains(t, err, "ocb init --from-config needs access to the Go module proxy")
		})
	}
}

func TestVerifyListError(t *testing.T) {
	list := func(context.Context, []string) ([]ListedModule, error) {
		return nil, assert.AnError
	}
	_, err := verify(t.Context(), list, []Component{{Receiver, "otlp"}}, acceptAll)
	require.ErrorIs(t, err, assert.AnError)
}

func TestVerifyNoComponents(t *testing.T) {
	proxy := &fakeProxy{}
	got, err := verify(t.Context(), proxy.list, nil, acceptAll)
	require.NoError(t, err)
	assert.Empty(t, got.modules)
	assert.Empty(t, proxy.calls)
}

func TestChunkArgs(t *testing.T) {
	long := strings.Repeat("a", 2000)
	chunks := chunkArgs([]string{long, long, long, long})
	require.Len(t, chunks, 2)
	assert.Len(t, chunks[0], 2)
	assert.Len(t, chunks[1], 2)

	assert.Empty(t, chunkArgs(nil))
	assert.Equal(t, [][]string{{"a", "b"}}, chunkArgs([]string{"a", "b"}))
}

func TestDecodeListed(t *testing.T) {
	got, err := decodeListed(strings.NewReader(`{"Path":"a","Versions":["v0.1.0","v0.2.0"]}
{"Path":"b","Error":{"Err":"404 Not Found"}}
{"Path":"c"}`))
	require.NoError(t, err)
	require.Len(t, got, 3)
	assert.True(t, got[0].exists())
	assert.False(t, got[1].exists())
	assert.Equal(t, "404 Not Found", got[1].Error.Err)
	assert.False(t, got[2].exists())

	_, err = decodeListed(strings.NewReader(`{"Path":`))
	require.Error(t, err)
}

func TestGoRunnerModule(t *testing.T) {
	g, err := newGoRunner()
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(g.dir, "go.mod"))

	require.NoError(t, g.close())
	assert.NoDirExists(t, g.dir)
}
