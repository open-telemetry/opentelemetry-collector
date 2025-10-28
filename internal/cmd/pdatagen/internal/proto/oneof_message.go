// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package proto // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/proto"

import (
	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/template"
)

const oneOfMessageOrigOtherTemplate = `
type {{ .oneOfMessageName }} struct {
	{{ .fieldName }} {{ .goType }}
}

func (m *{{ .parentMessageName }}) Get{{ .fieldName }}() {{ .goType }} {
	if v, ok := m.Get{{ .oneOfGroup }}().(*{{ .oneOfMessageName }}); ok {
		return v.{{ .fieldName }}
	}
	return {{ .defaultValue }}
}
`

const oneOfMessageOrigMessageTemplate = `
type {{ .oneOfMessageName }} struct {
	{{ .fieldName }} *{{ .goType }}
}

func (m *{{ .parentMessageName }}) Get{{ .fieldName }}() *{{ .goType }} {
	if v, ok := m.Get{{ .oneOfGroup }}().(*{{ .oneOfMessageName }}); ok {
		return v.{{ .fieldName }}
	}
	return nil
}
`

func (pf *Field) GenOneOfMessages() string {
	tf := pf.getTemplateFields()
	if pf.OneOfGroup != "" {
		if pf.Type == TypeMessage {
			return template.Execute(template.Parse("oneOfMessageOrigMessageTemplate", []byte(oneOfMessageOrigMessageTemplate)), tf)
		}
		return template.Execute(template.Parse("oneOfMessageOrigOtherTemplate", []byte(oneOfMessageOrigOtherTemplate)), tf)
	}
	return ""
}
