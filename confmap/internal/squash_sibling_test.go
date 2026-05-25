// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"testing"
)

// ExportedInnerWithUnmarshal is an exported type with a custom Unmarshal method,
// used to test that sibling fields are decoded correctly when this type is
// anonymously squash-embedded in an outer struct.
type ExportedInnerWithUnmarshal struct {
	Foo string `mapstructure:"foo"`
}

func (e *ExportedInnerWithUnmarshal) Unmarshal(c *Conf) error {
	return c.Unmarshal(e, WithIgnoreUnused())
}

type outerWithExportedSiblings struct {
	ExportedInnerWithUnmarshal `mapstructure:",squash"`
	Bar                        string `mapstructure:"bar"`
}

// TestSquashWithSiblingFields verifies that when an exported struct implementing
// Unmarshaler is anonymously squash-embedded in another struct, sibling fields of
// the outer struct are also decoded correctly.
func TestSquashWithSiblingFields(t *testing.T) {
	c := NewFromStringMap(map[string]any{
		"foo": "hello",
		"bar": "world",
	})
	cfg := &outerWithExportedSiblings{}
	err := c.Unmarshal(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Foo != "hello" {
		t.Errorf("Foo: got %q, want %q", cfg.Foo, "hello")
	}
	if cfg.Bar != "world" {
		t.Errorf("Bar: got %q, want %q", cfg.Bar, "world")
	}
}
