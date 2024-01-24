// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package zpagesextension

import (
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/extension/extensiontest"
)

func TestZPagesRequest(t *testing.T) {
	zh := &zpagesHandler{
		createSettings: extensiontest.NewNopCreateSettings(),
		host:           componenttest.NewNopHost(),
	}
	recorder := httptest.NewRecorder()

	zh.zPagesRequest(recorder, httptest.NewRequest("GET", "/", strings.NewReader("")))

	assert.Equal(t, 200, recorder.Result().StatusCode)
	b, err := io.ReadAll(recorder.Result().Body)
	assert.NoError(t, err)
	zpagesRegex, err := os.ReadFile(filepath.Join("testdata", "zpages.html"))
	assert.NoError(t, err)
	assert.Regexp(t, string(zpagesRegex), string(b))
}

func TestHandleServiceExtensions_Empty(t *testing.T) {
	zh := &zpagesHandler{
		createSettings: extensiontest.NewNopCreateSettings(),
		host:           componenttest.NewNopHost(),
	}
	recorder := httptest.NewRecorder()

	zh.handleServiceExtensions(recorder, httptest.NewRequest("GET", "/", strings.NewReader("")))

	assert.Equal(t, 200, recorder.Result().StatusCode)
	b, err := io.ReadAll(recorder.Result().Body)
	assert.NoError(t, err)
	extensions, err := os.ReadFile(filepath.Join("testdata", "extensions.html"))
	assert.NoError(t, err)
	assert.Equal(t, string(extensions), string(b))
}

func TestHandleFeaturezRequest(t *testing.T) {
	zh := &zpagesHandler{
		createSettings: extensiontest.NewNopCreateSettings(),
		host:           componenttest.NewNopHost(),
	}
	recorder := httptest.NewRecorder()

	zh.handleFeaturezRequest(recorder, httptest.NewRequest("GET", "/", strings.NewReader("")))

	assert.Equal(t, 200, recorder.Result().StatusCode)
	b, err := io.ReadAll(recorder.Result().Body)
	assert.NoError(t, err)
	features, err := os.ReadFile(filepath.Join("testdata", "features.html"))
	assert.NoError(t, err)
	assert.Equal(t, string(features), string(b))
}

type graphedHost struct {
	component.Host
}

func (g graphedHost) GetGraph() []struct {
	FullName    string
	InputType   string
	MutatesData bool
	Receivers   []string
	Processors  []string
	Exporters   []string
} {
	return []struct {
		FullName    string
		InputType   string
		MutatesData bool
		Receivers   []string
		Processors  []string
		Exporters   []string
	}{
		{
			FullName:    "foo",
			InputType:   "metrics",
			MutatesData: false,
			Receivers:   []string{"otlp", "file"},
			Processors:  []string{"batch"},
			Exporters:   []string{"otlp", "otlphttp"},
		},
	}
}

func TestHandlePipelinezRequest(t *testing.T) {
	g := graphedHost{
		Host: componenttest.NewNopHost(),
	}

	zh := &zpagesHandler{
		createSettings: extensiontest.NewNopCreateSettings(),
		host:           g,
	}
	recorder := httptest.NewRecorder()

	zh.handlePipelinezRequest(recorder, httptest.NewRequest("GET", "/?pipelinenamez=foo&componentnamez=batch&componentkindz=processor", strings.NewReader("")))

	assert.Equal(t, 200, recorder.Result().StatusCode)
	b, err := io.ReadAll(recorder.Result().Body)
	assert.NoError(t, err)
	pipelines, err := os.ReadFile(filepath.Join("testdata", "pipelines.html"))
	assert.NoError(t, err)
	assert.Equal(t, string(pipelines), string(b))
}
