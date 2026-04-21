// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package proto // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/proto"

import (
	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/tmplutil"
)

const poolVarOrigTemplate = `
	ProtoPool{{ .oneOfMessageName }} = sync.Pool{
		New: func() any {
			return &{{ .oneOfMessageName }}{}
		},
	}
`

func (pf *Field) GenPool() string {
	tf := pf.getTemplateFields()
	if pf.OneOfGroup != "" {
		return tmplutil.Execute(tmplutil.Parse("poolVarOrigTemplate", []byte(poolVarOrigTemplate)), tf)
	}
	return ""
}
