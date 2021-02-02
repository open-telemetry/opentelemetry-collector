// Copyright 2020 OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package scaffold

const Components = `
// Copyright 2020 OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenterror"
	{{- if .Distribution.IncludeCore}}
	"go.opentelemetry.io/collector/service/defaultcomponents"
	{{- end}}

	// extensions
	{{- range .Extensions}}
	{{.Name}} "{{.Import}}"
	{{- end}}

	// receivers 
	{{- range .Receivers}}
	{{.Name}} "{{.Import}}"
	{{- end}}

	// exporters
	{{- range .Exporters}}
	{{.Name}} "{{.Import}}"
	{{- end}}

	// processors
	{{- range .Processors}}
	{{.Name}} "{{.Import}}"
	{{- end}}
)

func components() (component.Factories, error) {
	var errs []error
	var err error
	var factories component.Factories

	{{- if .Distribution.IncludeCore}}
	factories, err = defaultcomponents.Components()
	if err != nil {
		return component.Factories{}, err
	}
	{{- else}}
	factories = component.Factories{}
	{{- end}}

	extensions := []component.ExtensionFactory{
		{{- range .Extensions}}
		{{.Name}}.NewFactory(),
		{{- end}}
	}
	for _, ext := range factories.Extensions {
		extensions = append(extensions, ext)
	}
	factories.Extensions, err = component.MakeExtensionFactoryMap(extensions...)
	if err != nil {
		errs = append(errs, err)
	}

	receivers := []component.ReceiverFactory{
		{{- range .Receivers}}
		{{.Name}}.NewFactory(),
		{{- end}}
	}
	for _, rcv := range factories.Receivers {
		receivers = append(receivers, rcv)
	}
	factories.Receivers, err = component.MakeReceiverFactoryMap(receivers...)
	if err != nil {
		errs = append(errs, err)
	}

	exporters := []component.ExporterFactory{
		{{- range .Exporters}}
		{{.Name}}.NewFactory(),
		{{- end}}
	}
	for _, exp := range factories.Exporters {
		exporters = append(exporters, exp)
	}
	factories.Exporters, err = component.MakeExporterFactoryMap(exporters...)
	if err != nil {
		errs = append(errs, err)
	}

	processors := []component.ProcessorFactory{
		{{- range .Processors}}
		{{.Name}}.NewFactory(),
		{{- end}}
	}
	for _, pr := range factories.Processors {
		processors = append(processors, pr)
	}
	factories.Processors, err = component.MakeProcessorFactoryMap(processors...)
	if err != nil {
		errs = append(errs, err)
	}

	return factories, componenterror.CombineErrors(errs)
}
`
