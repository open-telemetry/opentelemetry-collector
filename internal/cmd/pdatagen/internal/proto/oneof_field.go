package proto

import (
	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/template"
)

const oneOfTestFailingUnmarshalProtoValuesTemplate = `
	{{ range .Fields -}}
	{{ .GenTestFailingUnmarshalProtoValues }}
	{{- end }}`

const oneOfTestValuesTemplate = `
	{{ range .Fields -}}
	{{ .GenTestEncodingValues }}
	{{- end }}`

const oneOfDeleteOrigTemplate = `switch orig.{{ .GroupName }}Type() {
	{{ range .Fields -}}
	case {{ $.Name }}{{ $.GroupName }}Type{{ .GetName }}:
		{{ .GenDelete }}
	{{ end -}}
}`

const oneOfCopyOrigTemplate = `switch src.{{ .GroupName }}Type() {
	case {{ $.Name }}{{ $.GroupName }}TypeEmpty:
		dest.Reset{{ .GroupName }}()
	{{ range .Fields -}}
	case {{ $.Name }}{{ $.GroupName }}Type{{ .GetName }}:
		{{ .GenCopy }}
	{{ end -}}
}`

const oneOfMarshalJSONTemplate = `switch orig.{{ .GroupName }}Type() {
	{{ range .Fields -}}
	case {{ $.Name }}{{ $.GroupName }}Type{{ .GetName }}:
		{{ .GenMarshalJSON }}
	{{ end -}}
}`

const oneOfUnmarshalJSONTemplate = `
	{{ range .Fields -}}
		{{ .GenUnmarshalJSON }}
	{{- end }}`

const oneOfSizeProtoTemplate = `switch orig.{{ .GroupName }}Type() {
	case {{ .Name }}{{ .GroupName }}TypeEmpty:
		break
	{{ range .Fields -}}
	case {{ $.Name }}{{ $.GroupName }}Type{{ .GetName }}:
		{{ .GenSizeProto }}
	{{ end -}}
}`

const oneOfMarshalProtoTemplate = `switch orig.{{ .GroupName }}Type() {
	{{ range .Fields -}}
	case {{ $.Name }}{{ $.GroupName }}Type{{ .GetName }}:
		{{ .GenMarshalProto }}
	{{ end -}}
}`

const oneOfUnmarshalProtoTemplate = `
	{{- range .Fields }}
		{{ .GenUnmarshalProto }}
	{{ end }}`

type OneOfField struct {
	GroupName string
	Name      string
	Fields    []*Field
}

func (of *OneOfField) GenMessageField() string {
	return of.GroupName + " proto.OneOf"
}

func (of *OneOfField) GetName() string {
	return of.GroupName
}

func (of *OneOfField) GoType() string {
	panic("implement me")
}

func (of *OneOfField) DefaultValue() string {
	panic("implement me")
}

func (of *OneOfField) GenTest() string {
	return "orig.Set" + of.Fields[0].GetName() + "(" + of.Fields[0].TestValue() + ")"
}

func (of *OneOfField) TestValue() string {
	return of.Fields[0].TestValue()
}

func (of *OneOfField) GenTestFailingUnmarshalProtoValues() string {
	return template.Execute(template.Parse("oneOfTestFailingUnmarshalProtoValuesTemplate", []byte(oneOfTestFailingUnmarshalProtoValuesTemplate)), of)
}

func (of *OneOfField) GenTestEncodingValues() string {
	return template.Execute(template.Parse("oneOfTestValuesTemplate", []byte(oneOfTestValuesTemplate)), of)
}

func (of *OneOfField) GenDelete() string {
	return template.Execute(template.Parse("oneOfDeleteOrigTemplate", []byte(oneOfDeleteOrigTemplate)), of)
}

func (of *OneOfField) GenCopy() string {
	return template.Execute(template.Parse("oneOfCopyOrigTemplate", []byte(oneOfCopyOrigTemplate)), of)
}

func (of *OneOfField) GenMarshalJSON() string {
	return template.Execute(template.Parse("oneOfMarshalJSONTemplate", []byte(oneOfMarshalJSONTemplate)), of)
}

func (of *OneOfField) GenUnmarshalJSON() string {
	return template.Execute(template.Parse("oneOfUnmarshalJSONTemplate", []byte(oneOfUnmarshalJSONTemplate)), of)
}

func (of *OneOfField) GenSizeProto() string {
	t := template.Parse("oneOfSizeProtoTemplate", []byte(oneOfSizeProtoTemplate))
	return template.Execute(t, of)
}

func (of *OneOfField) GenMarshalProto() string {
	t := template.Parse("oneOfMarshalProtoTemplate", []byte(oneOfMarshalProtoTemplate))
	return template.Execute(t, of)
}

func (of *OneOfField) GenUnmarshalProto() string {
	t := template.Parse("oneOfUnmarshalProtoTemplate", []byte(oneOfUnmarshalProtoTemplate))
	return template.Execute(t, of)
}
