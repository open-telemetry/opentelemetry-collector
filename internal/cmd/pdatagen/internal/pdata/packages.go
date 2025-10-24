// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pdata // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/pdata"

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/proto"
)

var nonInternalDeps = []string{
	`"go.opentelemetry.io/collector/pdata/internal"`,
	`"go.opentelemetry.io/collector/pdata/pcommon"`,
	`"go.opentelemetry.io/collector/pdata/plog"`,
	`"go.opentelemetry.io/collector/pdata/pmetric"`,
	`"go.opentelemetry.io/collector/pdata/pprofile"`,
	`"go.opentelemetry.io/collector/pdata/ptrace"`,
	`"go.opentelemetry.io/collector/pdata/xpdata"`,
}

// AllPackages is a list of all packages that needs to be generated.
var AllPackages = []*Package{
	pcommon,
	plog,
	plogotlp,
	pmetric,
	pmetricotlp,
	ptrace,
	ptraceotlp,
	pprofile,
	pprofileotlp,
	xpdataEntity,
	prequest,
}

// Package is a struct used to generate files.
type Package struct {
	info *PackageInfo
	// Can be any of sliceStruct, sliceOfValues, messageStruct.
	structs []baseStruct
	enums   []*proto.Enum
}

type PackageInfo struct {
	name        string
	path        string
	imports     []string
	testImports []string
}

// GenerateFiles generates files with the configured data structures for this Package.
func (p *Package) GenerateFiles() error {
	for _, s := range p.structs {
		if s.getHasOnlyInternal() {
			continue
		}
		path := filepath.Join("pdata", p.info.path, "generated_"+strings.ToLower(s.getName())+".go")
		if err := os.WriteFile(path, s.generate(p.info), 0o600); err != nil {
			return err
		}
	}
	return nil
}

// GenerateTestFiles generates files with tests for the configured data structures for this Package.
func (p *Package) GenerateTestFiles() error {
	for _, s := range p.structs {
		if s.getHasOnlyInternal() {
			continue
		}
		path := filepath.Join("pdata", p.info.path, "generated_"+strings.ToLower(s.getName())+"_test.go")
		if err := os.WriteFile(path, s.generateTests(p.info), 0o600); err != nil {
			return err
		}
	}
	return nil
}

// GenerateInternalFiles generates files with internal structs for this Package.
func (p *Package) GenerateInternalFiles() error {
	for _, s := range p.structs {
		if !s.getHasWrapper() {
			continue
		}
		path := filepath.Join("pdata", "internal", "generated_wrapper_"+strings.ToLower(s.getOriginName())+".go")
		saveImports := slices.Clone(p.info.imports)
		p.info.imports = slices.DeleteFunc(p.info.imports, func(s string) bool {
			return slices.Contains(nonInternalDeps, s)
		})
		if err := os.WriteFile(path, s.generateInternal(p.info), 0o600); err != nil {
			return err
		}
		p.info.imports = saveImports
	}
	return nil
}

// GenerateProtoMessageFiles generates files with proto messages for this Package.
func (p *Package) GenerateProtoMessageFiles() error {
	for _, s := range p.structs {
		pm := s.getProtoMessage()
		if pm == nil {
			continue
		}
		saveTestImports := slices.Clone(p.info.testImports)
		p.info.testImports = slices.DeleteFunc(p.info.testImports, func(s string) bool {
			return slices.Contains(nonInternalDeps, s)
		})
		path := filepath.Join("pdata", "internal", "generated_proto_"+strings.ToLower(s.getOriginName())+".go")
		if err := os.WriteFile(path, pm.GenerateMessage(p.info.imports, p.info.testImports), 0o600); err != nil {
			return err
		}
		p.info.testImports = saveTestImports
	}
	return nil
}

// GenerateProtoMessageTestsFiles generates files with proto messages tests for this Package.
func (p *Package) GenerateProtoMessageTestsFiles() error {
	for _, s := range p.structs {
		pm := s.getProtoMessage()
		if pm == nil {
			continue
		}
		saveTestImports := slices.Clone(p.info.testImports)
		p.info.testImports = slices.DeleteFunc(p.info.testImports, func(s string) bool {
			return slices.Contains(nonInternalDeps, s)
		})
		path := filepath.Join("pdata", "internal", "generated_proto_"+strings.ToLower(pm.Name)+"_test.go")
		if err := os.WriteFile(path, pm.GenerateMessageTests(p.info.imports, p.info.testImports), 0o600); err != nil {
			return err
		}
		p.info.testImports = saveTestImports
	}
	return nil
}

// GenerateProtoEnumFiles generates files with proto messages for this Package.
func (p *Package) GenerateProtoEnumFiles() error {
	for _, s := range p.enums {
		path := filepath.Join("pdata", "internal", "generated_enum_"+strings.ToLower(s.Name)+".go")
		if err := os.WriteFile(path, s.GenerateEnum(), 0o600); err != nil {
			return err
		}
	}
	return nil
}

// usedByOtherDataTypes defines if the package is used by other data types and orig fields of the package's structs
// need to be accessible from other pdata packages.
func usedByOtherDataTypes(packageName string) bool {
	return packageName == "pcommon" || packageName == "entity"
}
