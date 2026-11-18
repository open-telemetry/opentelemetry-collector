// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package proto // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/proto"

import (
	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/tmplutil"
)

const oneOfMessageOrigBaseTemplate = `
var (
	protoPool{{ .oneOfMessageName }} = sync.Pool{
	    New: func() any {
		    return &{{ .oneOfMessageName }}{}
	    },
    }
)

`

const oneOfMessageOrigOtherTemplate = `
type {{ .oneOfMessageName }} struct {
	{{ .fieldName }} {{ .goType }}
}

func New{{ .oneOfMessageName }}() *{{ .oneOfMessageName }} {
	if !metadata.PdataUseProtoPoolingFeatureGate.IsEnabled() {
		return &{{ .oneOfMessageName }}{}
	}
	return protoPool{{ .oneOfMessageName }}.Get().(*{{ .oneOfMessageName }})
}

func delete{{ .oneOfMessageName }}(v *{{ .oneOfMessageName }}) {
	// Called only when pool is enabled
	v.{{ .fieldName }} = {{ .defaultValue }}
	protoPool{{ .oneOfMessageName }}.Put(v)
}

func (m *{{ .parentMessageName }}) Get{{ .fieldName }}() {{ .goType }} {
	if v, ok := m.Get{{ .oneOfGroup }}().(*{{ .oneOfMessageName }}); ok {
		return v.{{ .fieldName }}
	}
	return {{ .defaultValue }}
}

func (m *{{ .parentMessageName }}) Set{{ .fieldName }}(val {{ .goType }}) {
	if m.{{ .oneOfGroup }} != nil {
		// Reuse message if possible, otherwise delete and create a new one.
		if v, ok := m.Get{{ .oneOfGroup }}().(*{{ .oneOfMessageName }}); ok {
			v.{{ .fieldName }} = val
			return
		}
		m.delete{{ .oneOfGroup }}()
	}
	v := New{{ .oneOfMessageName }}();
	v.{{ .fieldName }} = val
	m.{{ .oneOfGroup }} = v
}
`

const oneOfMessageOrigMessageTemplate = `
type {{ .oneOfMessageName }} struct {
	{{ .fieldName }} *{{ .goType }}
}

func New{{ .oneOfMessageName }}() *{{ .oneOfMessageName }} {
	if !metadata.PdataUseProtoPoolingFeatureGate.IsEnabled() {
		return &{{ .oneOfMessageName }}{}
	}
	return protoPool{{ .oneOfMessageName }}.Get().(*{{ .oneOfMessageName }})
}

func delete{{ .oneOfMessageName }}(v *{{ .oneOfMessageName }}) {
	// Called only when pool is enabled
	v.{{ .fieldName }} = nil
	protoPool{{ .oneOfMessageName }}.Put(v)
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
			return tmplutil.Execute(tmplutil.Parse("oneOfMessageOrigMessageTemplate", []byte(oneOfMessageOrigBaseTemplate+oneOfMessageOrigMessageTemplate)), tf)
		}
		return tmplutil.Execute(tmplutil.Parse("oneOfMessageOrigOtherTemplate", []byte(oneOfMessageOrigBaseTemplate+oneOfMessageOrigOtherTemplate)), tf)
	}
	return ""
}
