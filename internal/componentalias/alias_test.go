// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componentalias

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
)

func TestNewTypeAliasHolder(t *testing.T) {
	holder := NewTypeAliasHolder()
	require.NotNil(t, holder)

	alias := holder.DeprecatedAlias()
	assert.Equal(t, component.Type{}, alias)
	assert.Empty(t, alias.String())

	testType := component.MustNewType("test_alias")
	holder.SetDeprecatedAlias(testType)
	retrievedAlias := holder.DeprecatedAlias()
	assert.Equal(t, testType, retrievedAlias)
	assert.Equal(t, "test_alias", retrievedAlias.String())
}

type mockFactory struct {
	factoryType component.Type
	TypeAliasHolder
}

func (f *mockFactory) Type() component.Type {
	return f.factoryType
}

func (f *mockFactory) CreateDefaultConfig() component.Config {
	return nil
}

func TestValidateComponentType_ExactMatch(t *testing.T) {
	testType := component.MustNewType("test")
	factory := &mockFactory{
		factoryType:     testType,
		TypeAliasHolder: NewTypeAliasHolder(),
	}

	testID := component.MustNewID(testType.String())
	err := ValidateComponentType(factory, testID)
	require.NoError(t, err)
}

func TestValidateComponentType_AliasMatch(t *testing.T) {
	factoryType := component.MustNewType("new_name")
	aliasType := component.MustNewType("old_name")

	factory := &mockFactory{
		factoryType:     factoryType,
		TypeAliasHolder: NewTypeAliasHolder(),
	}
	factory.SetDeprecatedAlias(aliasType)

	// Test with alias type
	aliasID := component.MustNewID(aliasType.String())
	err := ValidateComponentType(factory, aliasID)
	require.NoError(t, err)

	// Test with factory type still works
	factoryID := component.MustNewID(factoryType.String())
	err = ValidateComponentType(factory, factoryID)
	require.NoError(t, err)
}

func TestValidateComponentType_NoMatch(t *testing.T) {
	factoryType := component.MustNewType("factory_type")
	wrongType := component.MustNewType("wrong_type")

	factory := &mockFactory{
		factoryType:     factoryType,
		TypeAliasHolder: NewTypeAliasHolder(),
	}

	wrongID := component.MustNewID(wrongType.String())
	err := ValidateComponentType(factory, wrongID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "component type mismatch")
	assert.Contains(t, err.Error(), wrongType.String())
	assert.Contains(t, err.Error(), factoryType.String())
}

func TestValidateComponentType_NoMatchWithAlias(t *testing.T) {
	factoryType := component.MustNewType("factory_type")
	aliasType := component.MustNewType("alias_type")
	wrongType := component.MustNewType("wrong_type")

	factory := &mockFactory{
		factoryType:     factoryType,
		TypeAliasHolder: NewTypeAliasHolder(),
	}
	factory.SetDeprecatedAlias(aliasType)

	wrongID := component.MustNewID(wrongType.String())
	err := ValidateComponentType(factory, wrongID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "component type mismatch")
	assert.Contains(t, err.Error(), wrongType.String())
	assert.Contains(t, err.Error(), factoryType.String())
	assert.Contains(t, err.Error(), "deprecated alias type")
	assert.Contains(t, err.Error(), aliasType.String())
}

func TestValidateComponentType_EmptyAlias(t *testing.T) {
	factoryType := component.MustNewType("factory_type")
	wrongType := component.MustNewType("wrong_type")

	factory := &mockFactory{
		factoryType:     factoryType,
		TypeAliasHolder: NewTypeAliasHolder(),
	}
	// Don't set any alias (empty by default)

	wrongID := component.MustNewID(wrongType.String())
	err := ValidateComponentType(factory, wrongID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "component type mismatch")
	assert.NotContains(t, err.Error(), "deprecated alias type")
}

type mockFactoryWithoutAlias struct {
	factoryType component.Type
}

func (f *mockFactoryWithoutAlias) Type() component.Type {
	return f.factoryType
}

func (f *mockFactoryWithoutAlias) CreateDefaultConfig() component.Config {
	return nil
}

func TestValidateComponentType_FactoryWithoutAliasSupport(t *testing.T) {
	factoryType := component.MustNewType("factory_type")
	factory := &mockFactoryWithoutAlias{factoryType: factoryType}

	factoryID := component.MustNewID(factoryType.String())
	err := ValidateComponentType(factory, factoryID)
	require.NoError(t, err)

	wrongType := component.MustNewType("wrong_type")
	wrongID := component.MustNewID(wrongType.String())
	err = ValidateComponentType(factory, wrongID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "component type mismatch")
	assert.NotContains(t, err.Error(), "deprecated alias type")
}
