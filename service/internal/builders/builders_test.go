// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package builders

import (
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/internal/componentalias"
)

// mockFactory is a test factory that implements component.Factory
type mockFactory struct {
	factoryType component.Type
}

func (m *mockFactory) Type() component.Type {
	return m.factoryType
}

func (m *mockFactory) CreateDefaultConfig() component.Config {
	return nil
}

// mockFactoryWithAlias is a test factory that implements both component.Factory and componentalias.TypeAliasHolder
type mockFactoryWithAlias struct {
	factoryType component.Type
	aliasHolder componentalias.TypeAliasHolder
}

func (m *mockFactoryWithAlias) Type() component.Type {
	return m.factoryType
}

func (m *mockFactoryWithAlias) CreateDefaultConfig() component.Config {
	return nil
}

func (m *mockFactoryWithAlias) DeprecatedAlias() component.Type {
	return m.aliasHolder.DeprecatedAlias()
}

func (m *mockFactoryWithAlias) SetDeprecatedAlias(alias component.Type) {
	m.aliasHolder.SetDeprecatedAlias(alias)
}

func TestLogDeprecatedTypeAlias(t *testing.T) {
	tests := []struct {
		name          string
		factory       component.Factory
		usedType      component.Type
		expectWarning bool
	}{
		{
			name:          "no_alias_holder",
			factory:       &mockFactory{factoryType: component.MustNewType("test")},
			usedType:      component.MustNewType("test"),
			expectWarning: false,
		},
		{
			name: "no_alias_set",
			factory: &mockFactoryWithAlias{
				factoryType: component.MustNewType("test"),
				aliasHolder: componentalias.NewTypeAliasHolder(),
			},
			usedType:      component.MustNewType("test"),
			expectWarning: false,
		},
		{
			name: "using_current_type",
			factory: func() component.Factory {
				f := &mockFactoryWithAlias{
					factoryType: component.MustNewType("new"),
					aliasHolder: componentalias.NewTypeAliasHolder(),
				}
				f.aliasHolder.SetDeprecatedAlias(component.MustNewType("old"))
				return f
			}(),
			usedType:      component.MustNewType("new"),
			expectWarning: false,
		},
		{
			name: "using_deprecated_alias",
			factory: func() component.Factory {
				f := &mockFactoryWithAlias{
					factoryType: component.MustNewType("new"),
					aliasHolder: componentalias.NewTypeAliasHolder(),
				}
				f.aliasHolder.SetDeprecatedAlias(component.MustNewType("old"))
				return f
			}(),
			usedType:      component.MustNewType("old"),
			expectWarning: true,
		},
		{
			name: "using_unrelated_type",
			factory: func() component.Factory {
				f := &mockFactoryWithAlias{
					factoryType: component.MustNewType("new"),
					aliasHolder: componentalias.NewTypeAliasHolder(),
				}
				f.aliasHolder.SetDeprecatedAlias(component.MustNewType("old"))
				return f
			}(),
			usedType:      component.MustNewType("other"),
			expectWarning: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			core, logs := observer.New(zap.WarnLevel)
			logger := zap.New(core)

			logDeprecatedTypeAlias(logger, tt.factory, tt.usedType)

			if tt.expectWarning && logs.Len() != 1 {
				t.Errorf("expected 1 warning log but got %d", logs.Len())
			} else if !tt.expectWarning && logs.Len() > 0 {
				t.Errorf("expected no warning log but got %d", logs.Len())
			}
		})
	}
}
