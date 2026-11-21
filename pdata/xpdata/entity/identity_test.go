// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

func put(res pcommon.Resource, typ string, scope IdScope, scopeType string, id map[string]string) {
	e := ResourceEntities(res).PutEmpty(typ)
	e.SetIDScope(scope)
	e.SetIDScopeEntityType(scopeType)
	for k, v := range id {
		e.IdentifyingAttributes().PutStr(k, v)
	}
}

func TestResolveGlobalIdentity(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(pcommon.Resource)
		entityType string
		wantAttrs  map[string]any
		wantChain  []string
		wantReason UnresolvedReason
	}{
		{
			name: "global",
			setup: func(res pcommon.Resource) {
				put(res, "k8s.pod", IdScopeGlobal, "", map[string]string{"k8s.pod.uid": "p1"})
			},
			entityType: "k8s.pod",
			wantAttrs:  map[string]any{"k8s.pod.uid": "p1"},
			wantChain:  []string{"k8s.pod"},
			wantReason: Resolved,
		},
		{
			name: "two-level chain",
			setup: func(res pcommon.Resource) {
				put(res, "k8s.container", IdScopeLocal, "k8s.pod", map[string]string{"k8s.container.name": "c1"})
				put(res, "k8s.pod", IdScopeGlobal, "", map[string]string{"k8s.pod.uid": "p1"})
			},
			entityType: "k8s.container",
			wantAttrs:  map[string]any{"k8s.container.name": "c1", "k8s.pod.uid": "p1"},
			wantChain:  []string{"k8s.container", "k8s.pod"},
			wantReason: Resolved,
		},
		{
			name: "three-level chain, order of entities on the resource does not matter",
			setup: func(res pcommon.Resource) {
				put(res, "cloud", IdScopeGlobal, "", map[string]string{"cloud.provider": "gcp", "cloud.account.id": "proj"})
				put(res, "k8s.node", IdScopeLocal, "k8s.cluster", map[string]string{"k8s.node.name": "n1"})
				put(res, "k8s.cluster", IdScopeLocal, "cloud", map[string]string{"k8s.cluster.name": "c1"})
			},
			entityType: "k8s.node",
			wantAttrs:  map[string]any{"k8s.node.name": "n1", "k8s.cluster.name": "c1", "cloud.provider": "gcp", "cloud.account.id": "proj"},
			wantChain:  []string{"k8s.node", "k8s.cluster", "cloud"},
			wantReason: Resolved,
		},
		{
			name: "unspecified",
			setup: func(res pcommon.Resource) {
				put(res, "process", IdScopeUnspecified, "", map[string]string{"process.pid": "42"})
			},
			entityType: "process",
			wantAttrs:  map[string]any{"process.pid": "42"},
			wantChain:  []string{"process"},
			wantReason: UnresolvedScopeUnspecified,
		},
		{
			name: "unspecified in the middle of a chain",
			setup: func(res pcommon.Resource) {
				put(res, "k8s.container", IdScopeLocal, "k8s.pod", map[string]string{"k8s.container.name": "c1"})
				put(res, "k8s.pod", IdScopeUnspecified, "", map[string]string{"k8s.pod.name": "p1"})
			},
			entityType: "k8s.container",
			wantAttrs:  map[string]any{"k8s.container.name": "c1", "k8s.pod.name": "p1"},
			wantChain:  []string{"k8s.container", "k8s.pod"},
			wantReason: UnresolvedScopeUnspecified,
		},
		{
			name: "local with empty scope type is non-conforming",
			setup: func(res pcommon.Resource) {
				put(res, "k8s.container", IdScopeLocal, "", map[string]string{"k8s.container.name": "c1"})
			},
			entityType: "k8s.container",
			wantAttrs:  map[string]any{"k8s.container.name": "c1"},
			wantChain:  []string{"k8s.container"},
			wantReason: UnresolvedScopeEntityTypeEmpty,
		},
		{
			name: "scope entity missing",
			setup: func(res pcommon.Resource) {
				put(res, "k8s.namespace", IdScopeLocal, "k8s.cluster", map[string]string{"k8s.namespace.name": "ns"})
			},
			entityType: "k8s.namespace",
			wantAttrs:  map[string]any{"k8s.namespace.name": "ns"},
			wantChain:  []string{"k8s.namespace"},
			wantReason: UnresolvedScopeEntityMissing,
		},
		{
			name: "cycle",
			setup: func(res pcommon.Resource) {
				put(res, "a", IdScopeLocal, "b", map[string]string{"a.id": "1"})
				put(res, "b", IdScopeLocal, "a", map[string]string{"b.id": "2"})
			},
			entityType: "a",
			wantAttrs:  map[string]any{"a.id": "1", "b.id": "2"},
			wantChain:  []string{"a", "b"},
			wantReason: UnresolvedCyclic,
		},
		{
			name: "self scoped is a cycle",
			setup: func(res pcommon.Resource) {
				put(res, "a", IdScopeLocal, "a", map[string]string{"a.id": "1"})
			},
			entityType: "a",
			wantAttrs:  map[string]any{"a.id": "1"},
			wantChain:  []string{"a"},
			wantReason: UnresolvedCyclic,
		},
		{
			name: "dangling id key",
			setup: func(res pcommon.Resource) {
				put(res, "host", IdScopeGlobal, "", map[string]string{"host.id": "h"})
				res.Attributes().Remove("host.id")
			},
			entityType: "host",
			wantAttrs:  map[string]any{},
			wantChain:  []string{"host"},
			wantReason: UnresolvedIdentityIncomplete,
		},
		{
			name: "no id keys",
			setup: func(res pcommon.Resource) {
				put(res, "host", IdScopeGlobal, "", nil)
			},
			entityType: "host",
			wantAttrs:  map[string]any{},
			wantChain:  []string{"host"},
			wantReason: UnresolvedIdentityIncomplete,
		},
		{
			name: "global with a scope entity type is non-conforming",
			setup: func(res pcommon.Resource) {
				put(res, "host", IdScopeGlobal, "cloud", map[string]string{"host.id": "h"})
			},
			entityType: "host",
			wantAttrs:  map[string]any{"host.id": "h"},
			wantChain:  []string{"host"},
			wantReason: UnresolvedScopeEntityTypeUnexpected,
		},
		{
			name: "unknown scope value",
			setup: func(res pcommon.Resource) {
				put(res, "host", IdScope(7), "", map[string]string{"host.id": "h"})
			},
			entityType: "host",
			wantAttrs:  map[string]any{"host.id": "h"},
			wantChain:  []string{"host"},
			wantReason: UnresolvedScopeUnknown,
		},
		{
			name:       "no such entity",
			setup:      func(pcommon.Resource) {},
			entityType: "host",
			wantAttrs:  map[string]any{},
			wantChain:  nil,
			wantReason: UnresolvedNoSuchEntity,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := pcommon.NewResource()
			tt.setup(res)
			got := ResolveGlobalIdentity(res, tt.entityType)
			assert.Equal(t, tt.wantReason, got.Reason, got.Reason.String())
			assert.Equal(t, tt.wantReason == Resolved, got.Resolved())
			assert.Equal(t, tt.wantAttrs, got.Attributes.AsRaw())
			assert.Equal(t, tt.wantChain, got.Chain)
		})
	}
}

func TestIdScopeString(t *testing.T) {
	assert.Equal(t, "Unspecified", IdScopeUnspecified.String())
	assert.Equal(t, "Global", IdScopeGlobal.String())
	assert.Equal(t, "Local", IdScopeLocal.String())
	assert.Empty(t, IdScope(99).String())
}
