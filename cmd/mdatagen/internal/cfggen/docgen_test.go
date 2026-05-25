// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cfggen

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCfgPropDocs_DirectProperties(t *testing.T) {
	t.Parallel()
	cfg := &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"timeout":  {Type: "string", GoType: "time.Duration", Description: "request timeout", Default: "10s"},
			"endpoint": {Type: "string", Description: "endpoint URL"},
		},
		Required: []string{"endpoint"},
	}

	docs := CfgPropDocs(cfg)
	assert.Len(t, docs, 2)
	// sorted alphabetically
	assert.Equal(t, "endpoint", docs[0].Name)
	assert.True(t, docs[0].Required)
	assert.Equal(t, "timeout", docs[1].Name)
	assert.False(t, docs[1].Required)
}

func TestCfgPropDocs_FlattensAllOf(t *testing.T) {
	t.Parallel()
	embedded := &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"retry_on_failure": {Type: "boolean", Description: "retry on failure"},
		},
	}
	cfg := &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"endpoint": {Type: "string", Description: "endpoint URL"},
		},
		AllOf: []*ConfigMetadata{embedded},
	}

	docs := CfgPropDocs(cfg)
	assert.Len(t, docs, 2)
	names := make([]string, len(docs))
	for i, d := range docs {
		names[i] = d.Name
	}
	assert.Contains(t, names, "endpoint")
	assert.Contains(t, names, "retry_on_failure")
}

func TestCfgPropDocs_FlattensNestedAllOf(t *testing.T) {
	t.Parallel()
	// An embed that itself contains an AllOf
	inner := &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"level2": {Type: "string"},
		},
	}
	outer := &ConfigMetadata{
		Type:  "object",
		AllOf: []*ConfigMetadata{inner},
	}
	cfg := &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"top": {Type: "string"},
		},
		AllOf: []*ConfigMetadata{outer},
	}

	docs := CfgPropDocs(cfg)
	names := make([]string, len(docs))
	for i, d := range docs {
		names[i] = d.Name
	}
	assert.Contains(t, names, "top")
	assert.Contains(t, names, "level2")
}

func TestCfgPropDocs_NilReturnsNil(t *testing.T) {
	t.Parallel()
	assert.Nil(t, CfgPropDocs(nil))
}

func TestCfgIsObject(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		cfg    *ConfigMetadata
		expect bool
	}{
		{"nil", nil, false},
		{"primitive string", &ConfigMetadata{Type: "string"}, false},
		{"array of strings", &ConfigMetadata{Type: "array", Items: &ConfigMetadata{Type: "string"}}, false},
		{"inline object", &ConfigMetadata{Type: "object", Properties: map[string]*ConfigMetadata{"x": {Type: "string"}}}, true},
		{"ref-resolved object", &ConfigMetadata{Type: "object", ResolvedFrom: "confighttp.ServerConfig", Properties: map[string]*ConfigMetadata{"port": {Type: "integer"}}}, true},
		{"map without properties", &ConfigMetadata{Type: "object", AdditionalProperties: &ConfigMetadata{Type: "string"}}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expect, CfgIsObject(tc.cfg))
		})
	}
}

func TestCfgAnchor(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "endpoint", CfgAnchor("", "endpoint"))
	assert.Equal(t, "tls.ca_file", CfgAnchor("tls", "ca_file"))
	assert.Equal(t, "a.b.c", CfgAnchor("a.b", "c"))
}

func TestCfgDocType(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		cfg    *ConfigMetadata
		expect string
	}{
		{"nil", nil, "any"},
		{"string", &ConfigMetadata{Type: "string"}, "string"},
		{"duration via GoType", &ConfigMetadata{Type: "string", GoType: "time.Duration"}, "duration"},
		{"duration via Format", &ConfigMetadata{Type: "string", Format: "duration"}, "duration"},
		{"datetime via GoType", &ConfigMetadata{Type: "string", GoType: "time.Time"}, "datetime"},
		{"datetime via Format", &ConfigMetadata{Type: "string", Format: "date-time"}, "datetime"},
		{"enum string", &ConfigMetadata{Type: "string", Enum: []any{"a", "b"}}, "string (one of: a, b)"},
		{"integer", &ConfigMetadata{Type: "integer"}, "int"},
		{"number", &ConfigMetadata{Type: "number"}, "float"},
		{"boolean", &ConfigMetadata{Type: "boolean"}, "bool"},
		{"array of string", &ConfigMetadata{Type: "array", Items: &ConfigMetadata{Type: "string"}}, "[]string"},
		{"array of any", &ConfigMetadata{Type: "array"}, "[]any"},
		{"map of string", &ConfigMetadata{Type: "object", AdditionalProperties: &ConfigMetadata{Type: "string"}}, "map[string]string"},
		{"inline object", &ConfigMetadata{Type: "object", Properties: map[string]*ConfigMetadata{"x": {Type: "string"}}}, "object"},
		{"plain object no props", &ConfigMetadata{Type: "object"}, "object"},
		{"unknown type", &ConfigMetadata{Type: "unknown"}, "any"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expect, CfgDocType(tc.cfg))
		})
	}
}

func TestCfgDocDefault(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		cfg    *ConfigMetadata
		expect string
	}{
		{"nil cfg", nil, ""},
		{"nil default", &ConfigMetadata{Type: "string"}, ""},
		{"string default", &ConfigMetadata{Type: "string", Default: "localhost:4317"}, "localhost:4317"},
		{"bool default true", &ConfigMetadata{Type: "boolean", Default: true}, "true"},
		{"bool default false", &ConfigMetadata{Type: "boolean", Default: false}, "false"},
		{"int default", &ConfigMetadata{Type: "integer", Default: 42}, "42"},
		{"float default", &ConfigMetadata{Type: "number", Default: float64(3.14)}, "3.14"},
		{"slice default", &ConfigMetadata{Type: "array", Default: []any{"a", "b"}}, "[a, b]"},
		{"map default", &ConfigMetadata{Type: "object", Default: map[string]any{"key": "val"}}, "{key: val}"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expect, CfgDocDefault(tc.cfg))
		})
	}
}

func TestCfgRootSection(t *testing.T) {
	t.Parallel()
	cfg := &ConfigMetadata{Type: "object"}
	sec := CfgRootSection(cfg)
	assert.Equal(t, cfg, sec.Schema)
	assert.Empty(t, sec.Anchor)
	assert.Empty(t, sec.Title)
}

func TestCfgSubSection(t *testing.T) {
	t.Parallel()
	cfg := &ConfigMetadata{Type: "object"}
	sec := CfgSubSection(cfg, "tls", "tls")
	assert.Equal(t, cfg, sec.Schema)
	assert.Equal(t, "tls", sec.Anchor)
	assert.Equal(t, "tls", sec.Title)
}
