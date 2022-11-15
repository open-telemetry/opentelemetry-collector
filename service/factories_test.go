// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package service

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/extension"
)

func TestMakeExtensionFactoryMap(t *testing.T) {
	type testCase struct {
		name string
		in   []extension.Factory
		out  map[component.Type]extension.Factory
	}

	p1 := extension.NewExtensionFactory("p1", nil, nil, component.StabilityLevelAlpha)
	p2 := extension.NewExtensionFactory("p2", nil, nil, component.StabilityLevelAlpha)
	testCases := []testCase{
		{
			name: "different names",
			in:   []extension.Factory{p1, p2},
			out: map[component.Type]extension.Factory{
				p1.Type(): p1,
				p2.Type(): p2,
			},
		},
		{
			name: "same name",
			in:   []extension.Factory{p1, p2, extension.NewExtensionFactory("p1", nil, nil, component.StabilityLevelAlpha)},
		},
	}
	for i := range testCases {
		tt := testCases[i]
		t.Run(tt.name, func(t *testing.T) {
			out, err := MakeExtensionFactoryMap(tt.in...)
			if tt.out == nil {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.out, out)
		})
	}
}

func TestMakeReceiverFactoryMap(t *testing.T) {
	type testCase struct {
		name string
		in   []component.ReceiverFactory
		out  map[component.Type]component.ReceiverFactory
	}

	p1 := component.NewReceiverFactory("p1", nil)
	p2 := component.NewReceiverFactory("p2", nil)
	testCases := []testCase{
		{
			name: "different names",
			in:   []component.ReceiverFactory{p1, p2},
			out: map[component.Type]component.ReceiverFactory{
				p1.Type(): p1,
				p2.Type(): p2,
			},
		},
		{
			name: "same name",
			in:   []component.ReceiverFactory{p1, p2, component.NewReceiverFactory("p1", nil)},
		},
	}

	for i := range testCases {
		tt := testCases[i]
		t.Run(tt.name, func(t *testing.T) {
			out, err := MakeReceiverFactoryMap(tt.in...)
			if tt.out == nil {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.out, out)
		})
	}
}

func TestMakeProcessorFactoryMap(t *testing.T) {
	type testCase struct {
		name string
		in   []component.ProcessorFactory
		out  map[component.Type]component.ProcessorFactory
	}

	p1 := component.NewProcessorFactory("p1", nil)
	p2 := component.NewProcessorFactory("p2", nil)
	testCases := []testCase{
		{
			name: "different names",
			in:   []component.ProcessorFactory{p1, p2},
			out: map[component.Type]component.ProcessorFactory{
				p1.Type(): p1,
				p2.Type(): p2,
			},
		},
		{
			name: "same name",
			in:   []component.ProcessorFactory{p1, p2, component.NewProcessorFactory("p1", nil)},
		},
	}

	for i := range testCases {
		tt := testCases[i]
		t.Run(tt.name, func(t *testing.T) {
			out, err := MakeProcessorFactoryMap(tt.in...)
			if tt.out == nil {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.out, out)
		})
	}
}

func TestMakeExporterFactoryMap(t *testing.T) {
	type testCase struct {
		name string
		in   []exporter.Factory
		out  map[component.Type]exporter.Factory
	}

	p1 := exporter.NewFactory("p1", nil)
	p2 := exporter.NewFactory("p2", nil)
	testCases := []testCase{
		{
			name: "different names",
			in:   []exporter.Factory{p1, p2},
			out: map[component.Type]exporter.Factory{
				p1.Type(): p1,
				p2.Type(): p2,
			},
		},
		{
			name: "same name",
			in:   []exporter.Factory{p1, p2, exporter.NewFactory("p1", nil)},
		},
	}

	for i := range testCases {
		tt := testCases[i]
		t.Run(tt.name, func(t *testing.T) {
			out, err := MakeExporterFactoryMap(tt.in...)
			if tt.out == nil {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.out, out)
		})
	}
}
