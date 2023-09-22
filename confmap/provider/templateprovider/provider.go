// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package templateprovider // import "go.opentelemetry.io/collector/confmap/provider/templateprovider"

import (
	"context"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/internal"
)

const (
	schemeName  = "template"
	templateKey = "template"
	typeKey     = "type"

	// The template provider will always return the following structure:
	// {
	//   "templates": {
	//     "the_type_of_template": "the_template"
	//   },
	// }
	// This allows multiple templates to be aggreated under the same global key.
	allTemplatesKey = "templates"
)

type provider struct{}

// New returns a new confmap.Provider that reads the template from a file.
//
// This Provider supports "file" scheme, and can be called with a "uri" that follows:
//
//	file-uri		= "template:" local-path
//	local-path		= [ drive-letter ] file-path
//	drive-letter	= ALPHA ":"
//
// The "file-path" can be relative or absolute, and it can be any OS supported format.
//
// Examples:
// `template:path/to/template` - relative path (unix, windows)
// `template:/path/to/template` - absolute path (unix, windows)
// `template:c:/path/to/template` - absolute path including drive-letter (windows)
// `template:c:\path\to\template` - absolute path including drive-letter (windows)
func New() confmap.Provider {
	return &provider{}
}

func (fmp *provider) Retrieve(_ context.Context, uri string, _ confmap.WatcherFunc) (*confmap.Retrieved, error) {
	if !strings.HasPrefix(uri, schemeName+":") {
		return nil, fmt.Errorf("%q uri is not supported by %q provider", uri, schemeName)
	}

	// Clean the path before using it.
	content, err := os.ReadFile(filepath.Clean(uri[len(schemeName)+1:]))
	if err != nil {
		return nil, fmt.Errorf("unable to read the file %v: %w", uri, err)
	}

	retrieved, err := internal.NewRetrievedFromYAML(content)
	if err != nil {
		return nil, fmt.Errorf("read template %v: %w", uri, err)
	}

	templateConf, err := retrieved.AsConf()
	if err != nil {
		return nil, err
	}

	if !templateConf.IsSet("type") {
		return nil, fmt.Errorf("template %v: must have a 'type'", uri)
	}
	templateType, ok := templateConf.Get("type").(string)
	if !ok {
		return nil, fmt.Errorf("template %v: 'type' must be a string", uri)
	}

	if !templateConf.IsSet("template") {
		return nil, fmt.Errorf("template %v: must have a 'template'", uri)
	}

	rawTemplate, ok := templateConf.Get("template").(string)
	if !ok {
		return nil, fmt.Errorf("template %v: 'template' must be a string", uri)
	}

	if _, err = template.New(templateType).Parse(rawTemplate); err != nil {
		return nil, fmt.Errorf("template %v: parse as text/template: %w", uri, err)
	}

	return confmap.NewRetrieved(map[string]any{
		allTemplatesKey: map[string]any{
			templateType: rawTemplate,
		},
	})
}

func (*provider) Scheme() string {
	return schemeName
}

func (*provider) Shutdown(context.Context) error {
	return nil
}
