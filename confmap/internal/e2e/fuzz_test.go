// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package e2etest

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// targetNested tests the following property:
// > Passing a value of type T directly through an environment variable
// > should be equivalent to passing it through a nested environment variable.
func targetNested[T any](t *testing.T, value string) {
	resolver := NewResolver(t, "types_expand.yaml")

	// Use os.Setenv so we can check the error and return instead of failing the fuzzing.
	os.Setenv("ENV", "${env:ENV2}") //nolint:usetesting
	defer os.Unsetenv("ENV")
	err := os.Setenv("ENV2", value) //nolint:usetesting
	defer os.Unsetenv("ENV2")
	if err != nil {
		return
	}
	confNested, errResolveNested := resolver.Resolve(context.Background())

	err = os.Setenv("ENV", value) //nolint:usetesting
	if err != nil {
		return
	}
	confSimple, errResolveSimple := resolver.Resolve(context.Background())
	require.Equal(t, errResolveNested, errResolveSimple)
	if errResolveNested != nil {
		return
	}

	var cfgNested targetConfig[T]
	errNested := confNested.Unmarshal(cfgNested)

	var cfgSimple targetConfig[T]
	errSimple := confSimple.Unmarshal(cfgSimple)

	require.Equal(t, errNested, errSimple)
	if errNested != nil {
		return
	}
	assert.Equal(t, cfgNested, cfgSimple)
}

// testStrings for fuzzing targets
var testStrings = []string{
	"123",
	"opentelemetry",
	"!!str 123",
	"\"0123\"",
	"\"",
	"1111:1111:1111:1111:1111::",
	"{field: value}",
	"0xdeadbeef",
	"0b101",
	"field:",
	"2006-01-02T15:04:05Z07:00",
}

func FuzzNestedString(f *testing.F) {
	for _, value := range testStrings {
		f.Add(value)
	}
	f.Fuzz(targetNested[string])
}

func FuzzNestedInt(f *testing.F) {
	for _, value := range testStrings {
		f.Add(value)
	}
	f.Fuzz(targetNested[int])
}

func FuzzNestedMap(f *testing.F) {
	for _, value := range testStrings {
		f.Add(value)
	}
	f.Fuzz(targetNested[map[string]any])
}
