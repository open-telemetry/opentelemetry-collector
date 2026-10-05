// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ocbplugin // import "go.opentelemetry.io/collector/cmd/builder/ocbplugin"

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
	"golang.org/x/mod/semver"
)

// OCBPlugin defines the interface that plugins must implement.
type OCBPlugin interface {
	PreGenerate(config map[string]any) error
	PostGenerate(config map[string]any) error
	PreBuild(config map[string]any) error
	PostBuild(config map[string]any) error
	MinOCBVersion() string
}

type inputData struct {
	Action     string         `yaml:"action"`
	OCBVersion string         `yaml:"ocb_version"`
	Config     map[string]any `yaml:"config"`
}

// RunPlugin runs an OCBPlugin implementation. This should be called from main.
func RunPlugin(impl OCBPlugin) {
	// Currently plugins take no flags, but we want to do a flag parse and usage
	// to remain forward compatible in case we want flags in the future.
	flag.Usage = func() {
		_, _ = fmt.Fprintf(os.Stderr, "usage: %s <input-file>\n", filepath.Base(os.Args[0]))

		// If any flags ever get added, they will be part of the help message.
		flag.PrintDefaults()
	}
	flag.Parse()

	// If the plugin was run with no non-flag arguments, print usage and exit with code 2.
	// The intended calling path for a plugin is OCB itself, so if OCB calls the plugin
	// in a mistaken way then it needs to recognize that separately from a normal failure.
	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(2)
	}

	// The first argument should be a path to a plugin config file.
	if err := runPlugin(impl, flag.Arg(0)); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runPlugin(impl OCBPlugin, inputPath string) error {
	// We clean the filepath since it came from user input.
	// See gosec G304: https://securego.io/docs/rules/g304.html
	cleanInputPath := filepath.Clean(inputPath)
	inputBytes, err := os.ReadFile(cleanInputPath)
	if err != nil {
		return fmt.Errorf("error reading plugin input: %w", err)
	}
	if len(inputBytes) == 0 {
		return fmt.Errorf("the plugin input file at %s was empty", cleanInputPath)
	}
	var input inputData
	err = yaml.Unmarshal(inputBytes, &input)
	if err != nil {
		return fmt.Errorf("error decoding plugin input: %w", err)
	}
	err = checkSupportedVersion(impl, input.OCBVersion)
	if err != nil {
		return err
	}
	switch input.Action {
	case "pre-generate":
		err = impl.PreGenerate(input.Config)
	case "post-generate":
		err = impl.PostGenerate(input.Config)
	case "pre-build":
		err = impl.PreBuild(input.Config)
	case "post-build":
		err = impl.PostBuild(input.Config)
	default:
		err = fmt.Errorf("%w: %q", ErrUnknownAction, input.Action)
	}
	if err != nil {
		return fmt.Errorf("error running '%s' plugin action: %w", input.Action, err)
	}
	return nil
}

func checkSupportedVersion(impl OCBPlugin, ocbVersion string) error {
	// The minimum version the plugin supports.
	pluginMinVersion := impl.MinOCBVersion()

	// Normalize both semver strings by ensuring they are valid
	// and prepending the expected `v` prefix if it's not there already.
	ocbVersionNorm, err := normalizeSemverString(ocbVersion)
	if err != nil {
		return fmt.Errorf("couldn't normalize ocb version: %w", err)
	}
	pluginMinVersionNorm, err := normalizeSemverString(pluginMinVersion)
	if err != nil {
		return fmt.Errorf("couldn't normalize plugin-specified min ocb version: %w", err)
	}

	compare := semver.Compare(ocbVersionNorm, pluginMinVersionNorm)
	if compare < 0 {
		return fmt.Errorf(
			"%w: ocb version is %s but plugin requires at least %s",
			ErrUnsupportedOCBVersion,
			ocbVersion,
			pluginMinVersion,
		)
	}
	return nil
}

func normalizeSemverString(ver string) (string, error) {
	normalized := ver
	if !strings.HasPrefix(ver, "v") {
		normalized = "v" + ver
	}
	if !semver.IsValid(normalized) {
		return ver, fmt.Errorf("%w: %s (attempted to normalize to %s)", ErrInvalidSemverString, ver, normalized)
	}
	return normalized, nil
}

var (
	// ErrUnsupportedOCBVersion is returned when the running ocb version is too old for the plugin.
	ErrUnsupportedOCBVersion = errors.New("plugin does not support current ocb version")

	ErrInvalidSemverString = errors.New("invalid semver string")

	// ErrUnknownAction is returned when an action is requested of the plugin that is unrecognized.
	ErrUnknownAction = errors.New("unrecognized action")

	// ErrUnsupportedActionPreGenerate is returned when a plugin does not support the PreGenerate lifecycle hook action.
	ErrUnsupportedActionPreGenerate = errors.New("pre-generate action not supported")

	// ErrUnsupportedActionPostGenerate is returned when a plugin does not support the PostGenerate lifecycle hook action.
	ErrUnsupportedActionPostGenerate = errors.New("post-generate action not supported")

	// ErrUnsupportedActionPreBuild is returned when a plugin does not support the PreBuild lifecycle hook action.
	ErrUnsupportedActionPreBuild = errors.New("pre-build action not supported")

	// ErrUnsupportedActionPostBuild is returned when a plugin does not support the PostBuild lifecycle hook action.
	ErrUnsupportedActionPostBuild = errors.New("post-build action not supported")
)
