// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package schemagen // import "go.opentelemetry.io/collector/cmd/builder/internal/schemagen"

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/packages"
)

// CommentExtractor extracts field comments from Go source code using AST parsing.
type CommentExtractor struct {
	// commentCache maps package path -> "TypeName.FieldName" -> comment
	commentCache map[string]map[string]string
}

// NewCommentExtractor creates a new comment extractor.
func NewCommentExtractor() *CommentExtractor {
	return &CommentExtractor{
		commentCache: make(map[string]map[string]string),
	}
}

// ExtractComments parses all source files in a package to extract field comments.
func (ce *CommentExtractor) ExtractComments(pkg *packages.Package) {
	if _, exists := ce.commentCache[pkg.PkgPath]; exists {
		return
	}

	ce.commentCache[pkg.PkgPath] = make(map[string]string)

	for _, file := range pkg.Syntax {
		ce.extractCommentsFromFile(file, pkg.PkgPath)
	}
}

// GetFieldComment returns the comment for a specific field in a struct type.
func (ce *CommentExtractor) GetFieldComment(pkgPath, typeName, fieldName string) string {
	if packageComments, exists := ce.commentCache[pkgPath]; exists {
		key := typeName + "." + fieldName
		if comment, exists := packageComments[key]; exists {
			return comment
		}
	}
	return ""
}

func (ce *CommentExtractor) extractCommentsFromFile(file *ast.File, pkgPath string) {
	ast.Inspect(file, func(n ast.Node) bool {
		if node, ok := n.(*ast.TypeSpec); ok {
			if structType, ok := node.Type.(*ast.StructType); ok {
				ce.extractStructComments(node.Name.Name, structType, pkgPath)
			}
		}
		return true
	})
}

func (ce *CommentExtractor) extractStructComments(typeName string, structType *ast.StructType, pkgPath string) {
	for _, field := range structType.Fields.List {
		var comment string
		if field.Doc != nil {
			comment = cleanComment(field.Doc.Text())
		} else if field.Comment != nil {
			comment = cleanComment(field.Comment.Text())
		}

		for _, name := range field.Names {
			if comment != "" {
				key := typeName + "." + name.Name
				ce.commentCache[pkgPath][key] = comment
			}
		}
	}
}

// cleanComment removes comment markers and normalizes whitespace.
func cleanComment(comment string) string {
	comment = strings.TrimSpace(comment)
	lines := strings.Split(comment, "\n")
	var cleanedLines []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "//")
		line = strings.TrimPrefix(line, "/*")
		line = strings.TrimSuffix(line, "*/")
		line = strings.TrimSpace(line)
		if line != "" {
			cleanedLines = append(cleanedLines, line)
		}
	}

	return strings.Join(cleanedLines, " ")
}
