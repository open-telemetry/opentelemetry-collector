// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/cmd/schemagen/internal"

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type RunMode string

const (
	Component RunMode = "component"
	Package   RunMode = "package"
)

type Config struct {
	Mode              RunMode
	DirPath           string
	OutputFolder      string
	ConfigPackage     string
	ConfigType        string
	FileType          string
	Class             string
	Mappings          Mappings
	AllowedRefs       []string
	Namespace         string
	ResolveRefs       bool
	ComponentOverride *ComponentOverride
	SettingsDir       string
	Pattern           string
}

var (
	configType   = flag.String("c", "Config", "Config type name for component schema generation")
	outputFolder = flag.String("o", "", "Output schema folder (defaults to input folder)")
	fileType     = flag.String("t", "yaml", "Output file type (yaml or json)")
	resolveRefs  = flag.Bool("r", false, "Resolve external $ref entries inline in the output schema")
	pattern      = flag.String("p", ".", "Optional pattern to match config struct package, e.g. \"go.opentelemetry.io/collector/receiver/otlpreceiver")
)

func usage() {
	docs := []string{
		"Usage: schemagen [options] <path>",
		"This script is a tiny utility that walks a Go configuration file and emits JSON Schema that mirrors the exported structs.\n",
		"Options:\n",
		"\nArguments:",
		`  <input_file > Path to the dir to be processed. If not provided, the current working directory is used.`,
		"\nExamples:",
		"  > schemagen ./components/test_receiver/  		 # Generate schema for a component",
		"  > schemagen -o component.schema ./config.go       # Generate schema with a custom output file name",
		"  > schemagen -t json ./config.go                   # Generate schema in JSON format",
		"  > schemagen -c DatabaseConfig ./config.go         # Generate schema for component with a custom root type name",
	}
	_, _ = fmt.Fprintf(os.Stderr, "%s", strings.Join(docs[:3], "\n"))
	flag.PrintDefaults()
	_, _ = fmt.Fprintf(os.Stderr, "%s", strings.Join(docs[3:], "\n"))
}

func ReadConfig() (*Config, error) {
	flag.Usage = usage
	flag.Parse()

	inputPath := flag.Arg(0)
	if inputPath == "" {
		inputPath = "."
	}
	info, err := os.Stat(inputPath)
	if err != nil {
		return nil, err
	}

	var (
		dirPath       string
		output        = *outputFolder
		mode          = Package
		mappings      Mappings
		ctype         string
		class         string
		configPackage string
		allowedRefs   = make([]string, 0)
		namespace     string
	)

	switch {
	case info.IsDir():
		dirPath, _ = filepath.Abs(inputPath)
	default:
		dirPath, _ = filepath.Abs(filepath.Dir(inputPath))
	}

	if output == "" {
		output = dirPath
	}

	if *fileType != "json" && *fileType != "yaml" && *fileType != "yml" {
		return nil, errors.New("unknown schema file type - use yaml or json: " + *fileType)
	}

	if md, ok := ReadMetadata(dirPath); ok {
		if md.Parent != "" {
			mode = Component
		} else {
			ctype = md.Type
			class = md.Status.Class
			switch class {
			case "receiver", "processor", "exporter", "connector", "extension":
				mode = Component
			default:
				mode = Package
			}
		}
	}

	var (
		componentOverride *ComponentOverride
		settingsDir       string
	)
	if s, sDir, ok := ReadSettingsFile(); ok {
		mappings = s.Mappings
		settingsDir = sDir
		comp := class + "/" + ctype
		if override, found := s.ComponentOverrides[comp]; found {
			overrideCopy := override
			componentOverride = &overrideCopy
			if override.ConfigName != "" {
				*configType = override.ConfigName
			}
		}
		allowedRefs = s.AllowedRefs
		namespace = s.Namespace
	}

	configNameParts := strings.Split(*configType, ".")
	if len(configNameParts) == 2 {
		configPackage = configNameParts[0]
		*configType = configNameParts[1]
	}

	return &Config{
		DirPath:           dirPath,
		OutputFolder:      output,
		ConfigPackage:     configPackage,
		ConfigType:        *configType,
		FileType:          *fileType,
		Mode:              mode,
		Mappings:          mappings,
		Class:             class,
		AllowedRefs:       allowedRefs,
		Namespace:         namespace,
		ResolveRefs:       *resolveRefs,
		ComponentOverride: componentOverride,
		SettingsDir:       settingsDir,
		Pattern:           *pattern,
	}, nil
}

func (c *Config) Fork(mode RunMode, namespace string) *Config {
	cfg := *c
	cfg.Mode = mode
	cfg.Namespace = namespace
	cfg.ComponentOverride = nil
	return &cfg
}
