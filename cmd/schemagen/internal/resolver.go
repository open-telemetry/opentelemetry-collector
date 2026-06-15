// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/cmd/schemagen/internal"

import (
	"errors"
	"fmt"
	"log"
	"strings"
)

// NotFoundError is returned when a referenced type is not found in the target package.
type NotFoundError struct {
	TypeName    string
	PackageName string
}

func (e *NotFoundError) Error() string {
	if e.PackageName == "" {
		return fmt.Sprintf("reference type %q not found", e.TypeName)
	}
	return fmt.Sprintf("reference type %q not found in package %q", e.TypeName, e.PackageName)
}

func isNotFound(err error) (*NotFoundError, bool) {
	var nf *NotFoundError
	if errors.As(err, &nf) {
		return nf, true
	}
	return nil, false
}

type schemaLoader interface {
	Load(cfg *Config, pattern string) (*Schema, error)
}

type parserLoader struct{}

func (parserLoader) Load(cfg *Config, pattern string) (*Schema, error) {
	return NewParser(cfg).ParsePattern(pattern)
}

type RefResolver struct {
	rootCfg      *Config
	loader       schemaLoader
	packageCache map[string]*Schema
}

func NewRefResolver(cfg *Config) *RefResolver {
	return &RefResolver{rootCfg: cfg, loader: parserLoader{}, packageCache: make(map[string]*Schema)}
}

func (r *RefResolver) Resolve(schema *Schema) (*Schema, error) {
	return r.resolveSchema(schema, r.rootCfg, "")
}

func (r *RefResolver) resolveSchema(schema *Schema, cfg *Config, pkgName string) (*Schema, error) {
	for defName, def := range schema.Defs {
		e, err := r.resolveElement(schema, def, cfg, pkgName)
		if nf, ok := isNotFound(err); ok {
			log.Printf("schemagen: dropping def %q: %v", defName, nf)
			delete(schema.Defs, defName)
			continue
		}
		if err != nil {
			return nil, err
		}
		schema.Defs[defName] = e
	}

	object, err := r.resolveObject(schema, schema.ObjectSchemaElement, cfg, pkgName)
	if err != nil {
		return nil, err
	}
	schema.ObjectSchemaElement = *object

	// if component schema, cleanup Defs as they're already resolved inline
	if cfg.Mode == Component {
		schema.Defs = nil
	}

	return schema, nil
}

func (r *RefResolver) resolveObject(schema *Schema, object ObjectSchemaElement, cfg *Config, pkgName string) (*ObjectSchemaElement, error) {
	kept := object.AllOf[:0]
	for _, s := range object.AllOf {
		e, err := r.resolveElement(schema, s, cfg, pkgName)
		if nf, ok := isNotFound(err); ok {
			log.Printf("schemagen: dropping allOf entry: %v", nf)
			continue
		}
		if err != nil {
			return nil, err
		}
		kept = append(kept, e)
	}
	object.AllOf = kept

	for propName, prop := range object.Properties {
		e, err := r.resolveElement(schema, prop, cfg, pkgName)
		if nf, ok := isNotFound(err); ok {
			log.Printf("schemagen: dropping property %q: %v", propName, nf)
			delete(object.Properties, propName)
			continue
		}
		if err != nil {
			return nil, err
		}
		object.Properties[propName] = e
	}

	if object.AdditionalProperties != nil {
		a, err := r.resolveElement(schema, object.AdditionalProperties, cfg, pkgName)
		if nf, ok := isNotFound(err); ok {
			log.Printf("schemagen: dropping additionalProperties: %v", nf)
		} else if err != nil {
			return nil, err
		}

		object.AdditionalProperties = a
	}

	return &object, nil
}

func (r *RefResolver) resolveElement(schema *Schema, element SchemaElement, config *Config, parentPkg string) (SchemaElement, error) {
	switch e := element.(type) {
	case *ObjectSchemaElement:
		return r.resolveObject(schema, *e, config, parentPkg)
	case *ArraySchemaElement:
		resolvedItems, err := r.resolveElement(schema, e.Items, config, parentPkg)
		if err != nil {
			return nil, err // propagate NotFoundError so parent drops the array
		}
		e.Items = resolvedItems
		return element, nil
	case *RefSchemaElement:
		return r.resolveRef(schema, e, config, parentPkg)
	}

	return element, nil
}

func (r *RefResolver) resolveRef(schema *Schema, refObj *RefSchemaElement, config *Config, parentPkg string) (SchemaElement, error) {
	ref := NewRef(refObj.Ref, parentPkg, config)
	cfg := config.Fork(Package, ref.namespace)

	var typeSchema SchemaElement
	def, isLocal := schema.Defs[refObj.Ref]
	if isLocal && ref.packageName == "" {
		typeSchema = def
	} else {
		innerSchema, cached := r.packageCache[ref.packageName]
		if !cached {
			parsed, err := r.loader.Load(cfg, ref.packageName)
			if err != nil {
				return nil, err
			}
			// Register before recursing so cyclic refs short-circuit here.
			r.packageCache[ref.packageName] = parsed
			innerSchema = parsed
			if _, err := r.resolveSchema(parsed, cfg, ref.packageName); err != nil {
				return nil, err
			}
		}

		var found bool
		typeSchema, found = innerSchema.Defs[ref.typeName]
		if !found {
			return nil, &NotFoundError{TypeName: ref.typeName, PackageName: ref.packageName}
		}
	}
	// copy custom annotations from original element
	base := refObj.BaseSchemaElement
	typeSchema.setOptional(base.IsOptional)
	typeSchema.setIsPointer(base.IsPointer)
	typeSchema.setDescription(base.Description)

	return typeSchema, nil
}

type Ref struct {
	typeName    string
	packageName string
	namespace   string
}

func NewRef(refPath, parentPkg string, cfg *Config) *Ref {
	i := strings.LastIndex(refPath, ".")
	if i == -1 {
		return &Ref{typeName: refPath}
	}
	packageName := refPath[:i]
	typeName := refPath[i+1:]

	// resolve local packages
	switch {
	case strings.HasPrefix(packageName, "/"):
		packageName = cfg.Namespace + packageName
	case strings.HasPrefix(packageName, "./"):
		if parentPkg != "" {
			packageName = parentPkg + "/" + strings.TrimPrefix(packageName, "./")
		}
		// otherwise keep it as relative path, it will be resolved by packages.Load
	}

	// find out namespace
	namespace := ""
	for _, ns := range cfg.AllowedRefs {
		if strings.HasPrefix(packageName, ns) {
			namespace = ns
			break
		}
	}

	return &Ref{typeName: typeName, packageName: packageName, namespace: namespace}
}
