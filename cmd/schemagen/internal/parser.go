// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/cmd/schemagen/internal"

import (
	"container/list"
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/iancoleman/strcase"
	"golang.org/x/tools/go/packages"
)

type Parser struct {
	config       *Config
	schema       *Schema
	types        map[string]*TypeInfo
	processQueue *list.List
	pkg          *packages.Package
	current      *TypeInfo
}

type TypeInfo struct {
	spec      *ast.TypeSpec
	comms     []*ast.CommentGroup
	imports   map[string]string
	typeName  string
	processed bool
}

func NewParser(cfg *Config) *Parser {
	return &Parser{
		config:       cfg,
		types:        make(map[string]*TypeInfo),
		processQueue: list.New(),
	}
}

func (p *Parser) Parse() (*Schema, error) {
	return p.ParsePattern(p.config.Pattern)
}

func (p *Parser) ParsePattern(pattern string) (*Schema, error) {
	set := token.NewFileSet()
	pkgs, e := packages.Load(&packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedSyntax | packages.NeedModule,
		Fset:  set,
		Dir:   p.config.DirPath,
		Tests: false,
	}, pattern)

	if e != nil {
		return nil, e
	}

	p.pkg = pkgs[0]
	p.schema = CreateSchema()
	p.processPackages(set, pkgs)

	// select types to process
	if err := p.feedProcessQueue(); err != nil {
		return nil, err
	}

	// process types
	if err := p.processTypes(); err != nil {
		return nil, err
	}

	if err := p.expandFactoryMaps(); err != nil {
		return nil, err
	}

	return p.schema, nil
}

func (p *Parser) processPackages(set *token.FileSet, pkgs []*packages.Package) {
	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			p.collectTypesAndImports(file, pkg.PkgPath, ast.NewCommentMap(set, file, file.Comments))
		}
	}
}

func (p *Parser) collectTypesAndImports(file *ast.File, pkgPath string, cmap ast.CommentMap) {
	target := p.types
	// collect imports from current file, distinguish internal vs external
	imports := make(map[string]string)
	for _, imp := range file.Imports {
		path, name := ParseImport(imp)
		isInternal := strings.HasPrefix(path, pkgPath)
		if isInternal {
			path = "." + strings.TrimPrefix(path, pkgPath)
		}
		imports[name] = path
	}
	// collect exported type specs
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}
		comms := cmap[genDecl]
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			if typeSpec.Name.IsExported() {
				name := typeSpec.Name.Name
				target[name] = &TypeInfo{typeSpec, comms, imports, name, false}
			}
		}
	}
}

func (p *Parser) feedProcessQueue() error {
	configTypeName := p.config.ConfigType
	// in component mode process only the config type initially
	if p.config.Mode == Component {
		if typeInfo, ok := p.types[configTypeName]; ok {
			p.processQueue.PushBack(typeInfo)
		}
	} else { // in package mode process all exported types
		for _, typeInfo := range p.types {
			p.processQueue.PushBack(typeInfo)
		}
	}
	if len(p.types) == 0 {
		return errors.New("no exported types found in the package")
	}
	return nil
}

func (p *Parser) processTypes() error {
	for p.processQueue.Len() > 0 {
		// pick next type to process from the queue
		item := p.processQueue.Front()
		p.processQueue.Remove(item)

		typeInfo, _ := item.Value.(*TypeInfo)
		typeName := typeInfo.typeName
		// skip already processed types
		if typeInfo.processed {
			continue
		}
		typeInfo.processed = true
		p.current = typeInfo

		schemaElement, err := p.parseType(typeInfo)
		if err != nil {
			return fmt.Errorf("parse type spec %s: %w", typeName, err)
		}
		if schemaElement == nil {
			continue
		}

		if obj, ok := schemaElement.(*ObjectSchemaElement); ok {
			isEmpty := len(obj.Properties) == 0 && len(obj.AllOf) == 0
			if isEmpty {
				continue // skip struct types with no exported fields
			}
		}

		// add parsed type to schema
		if p.isConfigType(typeInfo) {
			if obj, ok := schemaElement.(*ObjectSchemaElement); ok {
				p.schema.ObjectSchemaElement = *obj
			}
			if field, ok := schemaElement.(*FieldSchemaElement); ok {
				p.schema.ElementType = field.ElementType
			}
			if ref, ok := schemaElement.(*RefSchemaElement); ok {
				p.schema.ElementType = SchemaTypeObject
				p.schema.AllOf = append(p.schema.AllOf, ref)
			}
		} else {
			p.schema.Defs.AddDef(strcase.ToSnake(typeName), schemaElement)
		}
	}
	return nil
}

func (p *Parser) isConfigType(typeInfo *TypeInfo) bool {
	return p.config.Mode == Component && typeInfo.typeName == p.config.ConfigType
}

func (p *Parser) parseType(typeInfo *TypeInfo) (SchemaElement, error) {
	typeSpec := typeInfo.spec
	// skip non-struct types at the top level
	switch typeSpec.Type.(type) {
	case *ast.InterfaceType, *ast.FuncType:
		// skip these types
		return nil, nil
	}
	schemaElement, err := p.parseExpr(typeSpec.Type)
	if err != nil {
		return nil, err
	}
	if schemaElement != nil && len(typeInfo.comms) > 0 {
		if desc, ok := ExtractDescriptionFromComment(typeInfo.comms[0]); ok {
			schemaElement.setDescription(desc)
		}
	}

	return schemaElement, nil
}

func (p *Parser) parseExpr(expr ast.Expr) (SchemaElement, error) {
	switch t := expr.(type) {
	case *ast.ArrayType:
		return p.parseArray(t)
	case *ast.Ident:
		return p.parseIdent(t)
	case *ast.StructType:
		return p.parseStruct(t)
	case *ast.MapType:
		return p.parseMap(t)
	case *ast.StarExpr:
		return p.parsePointer(t)
	case *ast.SelectorExpr:
		return p.parseSelector(t)
	case *ast.IndexExpr:
		return p.parseOptional(t)
	}

	return nil, errors.New("unrecognized field type" + fmt.Sprintf(" (%T)", expr))
}

func (p *Parser) parseStruct(structType *ast.StructType) (SchemaElement, error) {
	var schemaObject SchemaObject = CreateObjectField("")

	for _, field := range structType.Fields.List {
		tag, ok := ParseTag(field.Tag)
		if !ok {
			continue
		}
		if len(field.Names) == 0 || tag.Squash {
			if err := p.addEmbeddedField(field, schemaObject); err != nil {
				return nil, err
			}
			continue
		}
		p.addNamedFields(field, schemaObject)
	}

	return schemaObject.(SchemaElement), nil
}

func (p *Parser) addEmbeddedField(field *ast.Field, schemaObject SchemaObject) error {
	ident, ok := field.Type.(*ast.Ident)
	if !ok {
		selector, ok := field.Type.(*ast.SelectorExpr)
		if ok {
			element, err := p.parseSelector(selector)
			if err != nil {
				return err
			}
			schemaObject.AddEmbedded(element)
			return nil
		}

		return errors.New("unrecognized embedded field type ")
	}

	elem, err := p.parseIdent(ident)
	if err == nil {
		switch elem := elem.(type) {
		case *RefSchemaElement:
			schemaObject.AddEmbedded(elem)
			return nil
		case *ObjectSchemaElement:
			return mergeSchemas(schemaObject, elem)
		}
	}
	return err
}

func (p *Parser) addNamedFields(field *ast.Field, schemaObject SchemaObject) {
	for _, ident := range field.Names {
		tag, hasTag := ParseTag(field.Tag)
		isValid := ident.IsExported() && hasTag
		if !isValid {
			continue
		}
		fieldName := tag.Name
		if fieldName == "" {
			fieldName = ident.Name
		}
		p.addNamedField(fieldName, field, schemaObject)
	}
}

func (p *Parser) addNamedField(fieldName string, field *ast.Field, schemaObject SchemaObject) {
	element, err := p.parseExpr(field.Type)
	if err != nil {
		fmt.Printf("Error parsing field %s: %v\n", fieldName, err)
		return
	}

	if description, ok := ExtractDescriptionFromComment(field.Doc); ok {
		element.setDescription(description)
	}

	schemaObject.AddProperty(fieldName, element)
}

func (p *Parser) parseArray(array *ast.ArrayType) (SchemaElement, error) {
	itemSchema, err := p.parseExpr(array.Elt)
	if err != nil {
		return nil, err
	}
	return CreateArrayField(itemSchema, ""), nil
}

func (p *Parser) parseIdent(ident *ast.Ident) (SchemaElement, error) {
	typeName := ident.Name
	if primitiveType, isCustom := goPrimitiveToSchemaType(typeName); primitiveType != SchemaTypeUnknown {
		element := CreateSimpleField(primitiveType, "")
		if isCustom {
			element.CustomElementType = typeName
		}
		return element, nil
	}

	if info, exists := p.types[typeName]; exists {
		p.processQueue.PushBack(info)
		return CreateRefField(strcase.ToSnake(typeName), ""), nil
	}

	if ident.Obj != nil {
		if typeSpec, ok := ident.Obj.Decl.(*ast.TypeSpec); ok {
			elem, err := p.parseExpr(typeSpec.Type)
			if err != nil {
				return nil, err
			}
			return elem, nil
		}
	}

	return nil, fmt.Errorf("type %s not found in collected type specs", typeName)
}

func (p *Parser) parseMap(m *ast.MapType) (SchemaElement, error) {
	valueSchema, err := p.parseExpr(m.Value)
	if err != nil {
		return nil, err
	}
	return CreateMapField(valueSchema, ""), nil
}

func (p *Parser) parsePointer(pointer *ast.StarExpr) (SchemaElement, error) {
	element, err := p.parseExpr(pointer.X)
	if err != nil {
		return nil, err
	}
	element.setIsPointer(true)
	return element, nil
}

func (p *Parser) parseSelector(selector *ast.SelectorExpr) (SchemaElement, error) {
	pkgIdent, ok := selector.X.(*ast.Ident)
	if !ok {
		return nil, errors.New("unrecognized SelectorExpr structure")
	}

	pkgName := pkgIdent.Name
	name := selector.Sel.Name
	fullTypeName := pkgName + "." + name

	if path, ok := p.current.imports[pkgName]; ok {
		_, exists := p.config.Mappings[path]
		if !exists {
			// always allow internal packages
			allowed := strings.HasPrefix(path, ".")
			// otherwise check allowed refs
			if !allowed {
				for _, allowedPath := range p.config.AllowedRefs {
					if strings.HasPrefix(path, allowedPath) {
						allowed = true
						break
					}
				}
			}
			fullID := fmt.Sprintf("%s.%s", path, strcase.ToSnake(name))
			// if allowed - create ref, else create any with custom type
			if allowed {
				var refID string
				// if ref is in the same namespace/repository
				if path == p.config.Namespace || strings.HasPrefix(path, p.config.Namespace+"/") {
					refID, _ = strings.CutPrefix(fullID, p.config.Namespace)
				} else {
					refID = fullID
				}
				element := CreateRefField(refID, "")
				return element, nil
			}
			element := CreateSimpleField(SchemaTypeAny, "")
			element.CustomElementType = fullID
			return element, nil
		}
		pkgName = path
		fullTypeName = pkgName + "." + name
	}

	if pkg, ok := p.config.Mappings[pkgName]; ok {
		if typeDesc, ok := pkg[name]; ok {
			element := CreateSimpleField(typeDesc.SchemaType, "")
			if !typeDesc.SkipAnnotation {
				element.CustomElementType = fullTypeName
			}
			element.Format = typeDesc.Format
			return element, nil
		}
	}

	if info, exists := p.types[name]; exists {
		p.processQueue.PushBack(info)
		element := CreateRefField(strcase.ToSnake(name), "")
		return element, nil
	}

	return nil, fmt.Errorf("unrecognized type in selector: %s", fullTypeName)
}

func (p *Parser) parseOptional(indexExpr *ast.IndexExpr) (SchemaElement, error) {
	wrapperType, ok := indexExpr.X.(*ast.SelectorExpr)
	if !ok {
		return nil, errors.New("unrecognized IndexExpr structure")
	}
	wrapperTypeName := wrapperType.Sel.Name

	if wrapperTypeName == "Optional" {
		element, err := p.parseExpr(indexExpr.Index)
		if err == nil {
			element.setOptional(true)
		}
		return element, err
	}

	fmt.Printf("Warning: unrecognized generic type: %s\n", wrapperTypeName)

	return nil, nil
}
