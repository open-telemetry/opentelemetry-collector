// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package entity // import "go.opentelemetry.io/collector/pdata/xpdata/entity"

import (
	"go.opentelemetry.io/collector/pdata/internal"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

func ResourceEntityRefs(res pcommon.Resource) EntityRefSlice {
	ir := internal.Resource(res)
	return newEntityRefSlice(&internal.GetOrigResource(ir).EntityRefs, internal.GetResourceState(ir))
}
