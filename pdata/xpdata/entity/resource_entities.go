// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package entity // import "go.opentelemetry.io/collector/pdata/xpdata/entity"

import (
	"go.opentelemetry.io/collector/pdata/internal"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

// ResourceEntityRefs returns the EntityRefs associated with this Resource.
// Once EntityRefs is stabilized in the proto definition,
// this function will be available in the pcommon package as part of a Resource method.
func ResourceEntityRefs(res pcommon.Resource) EntityRefSlice {
	ir := internal.ResourceWrapper(res)
	return newEntityRefSlice(&internal.GetResourceOrig(ir).EntityRefs, internal.GetResourceState(ir))
}

// ResourceEntities returns the Entities associated with this Resource.
// The returned EntityMap shares the resource's attributes map, so modifications
// to entity attributes are immediately reflected in the resource.
func ResourceEntities(res pcommon.Resource) EntityMap {
	return EntityMap{
		refs:       ResourceEntityRefs(res),
		attributes: res.Attributes(),
	}
}
