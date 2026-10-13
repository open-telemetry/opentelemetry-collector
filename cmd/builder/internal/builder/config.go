// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package builder // import "go.opentelemetry.io/collector/cmd/builder/internal/builder"

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"time"

	"go.uber.org/multierr"
	"go.uber.org/zap"
)

const (
	DefaultBetaOtelColVersion   = "v0.162.0"
	DefaultStableOtelColVersion = "v1.68.0"
)

// errMissingGoMod indicates an empty gomod field
var errMissingGoMod = errors.New("missing gomod specification for module")

// errUnknownResourceDetector indicates a detector name configured under
// dist.resource.detection.detectors that is not in knownResourceDetectors.
var errUnknownResourceDetector = errors.New("unknown resource detector")

// Config holds the builder's configuration
type Config struct {
	Logger *zap.Logger `mapstructure:"-"`

	OtelColVersion       string `mapstructure:"-"` // only used be the go.mod template
	SkipGenerate         bool   `mapstructure:"-"`
	SkipCompilation      bool   `mapstructure:"-"`
	SkipGetModules       bool   `mapstructure:"-"`
	SkipStrictVersioning bool   `mapstructure:"-"`
	LDFlags              string `mapstructure:"-"`
	LDSet                bool   `mapstructure:"-"` // only used to override LDFlags
	GCFlags              string `mapstructure:"-"`
	GCSet                bool   `mapstructure:"-"` // only used to override GCFlags
	Verbose              bool   `mapstructure:"-"`

	Distribution      Distribution `mapstructure:"dist"`
	Exporters         []Module     `mapstructure:"exporters,omitempty"`
	Extensions        []Module     `mapstructure:"extensions,omitempty"`
	Receivers         []Module     `mapstructure:"receivers,omitempty"`
	Processors        []Module     `mapstructure:"processors,omitempty"`
	Connectors        []Module     `mapstructure:"connectors,omitempty"`
	Telemetry         Module       `mapstructure:"telemetry,omitempty"`
	ConfmapProviders  []Module     `mapstructure:"providers,omitempty"`
	ConfmapConverters []Module     `mapstructure:"converters,omitempty"`
	Replaces          []string     `mapstructure:"replaces,omitempty"`
	Excludes          []string     `mapstructure:"excludes,omitempty"`

	ConfResolver ConfResolver `mapstructure:"conf_resolver,omitempty"`

	downloadModules retry `mapstructure:"-"`
}

type ConfResolver struct {
	// When set, will be used to set the CollectorSettings.ConfResolver.DefaultScheme value,
	// which determines how the Collector interprets URIs that have no scheme, such as ${ENV}.
	// See https://pkg.go.dev/go.opentelemetry.io/collector/confmap#ResolverSettings for more details.
	DefaultURIScheme string `mapstructure:"default_uri_scheme,omitempty"`
}

// Distribution holds the parameters for the final binary
type Distribution struct {
	Module                  string   `mapstructure:"module,omitempty"`
	Name                    string   `mapstructure:"name"`
	Go                      string   `mapstructure:"go,omitempty"`
	Description             string   `mapstructure:"description"`
	OutputPath              string   `mapstructure:"output_path"`
	Version                 string   `mapstructure:"version,omitempty"`
	BuildTags               string   `mapstructure:"build_tags,omitempty"`
	DebugCompilation        bool     `mapstructure:"debug_compilation,omitempty"`
	CGoEnabled              bool     `mapstructure:"cgo_enabled,omitempty"`
	UseAbsoluteReplacePaths bool     `mapstructure:"use_absolute_replace_paths,omitempty"`
	Resource                Resource `mapstructure:"resource,omitempty"`
}

// Resource holds resource configuration for the distribution. Its shape mirrors the
// "resource" stanza of the OpenTelemetry configuration schema:
// https://github.com/open-telemetry/opentelemetry-configuration/blob/main/examples/otel-getting-started.yaml
type Resource struct {
	Detection ResourceDetection `mapstructure:"detection,omitempty"`
}

// ResourceDetection controls which resourcedetectionprocessor detectors are compiled
// into the distribution. When Detectors is empty, every detector is left in. When
// Detectors is non-empty, every known detector not listed is compiled out via a
// matching omit_detector_<name> go build tag.
type ResourceDetection struct {
	Detectors []ResourceDetector `mapstructure:"detectors,omitempty"`
}

// ResourceDetector is a single entry in a detectors list, keyed by the
// resourcedetectionprocessor detector name (e.g. "env", "system", "gcp"), matching the
// shape used by the OpenTelemetry configuration schema's detector list.
type ResourceDetector map[string]any

// omitDetectorBuildTag is the prefix of the go build tag that excludes a single
// resource detector's implementation from compilation, matching the convention used by
// resourcedetectionprocessor (opentelemetry-collector-contrib) and otelconf
// (opentelemetry-go-contrib), e.g. omit_detector_env, omit_detector_aws_ec2.
const omitDetectorBuildTag = "omit_detector_"

// knownResourceDetectors lists every detector name that currently supports being
// compiled out via a matching omit_detector_<name> build tag, across
// resourcedetectionprocessor (opentelemetry-collector-contrib) and otelconf
// (opentelemetry-go-contrib). It must be kept in sync with those repositories.
var knownResourceDetectors = []string{
	"akamai",
	"alibaba_ecs",
	"aws_ec2",
	"aws_ecs",
	"aws_eks",
	"aws_elastic_beanstalk",
	"aws_lambda",
	"azure",
	"azure_aks",
	"azure_appservice",
	"azure_containerapps",
	"azure_functions",
	"azure_vm",
	"consul",
	"digitalocean",
	"docker",
	"dynatrace",
	"env",
	"gcp",
	"heroku",
	"hetzner",
	"ibmcloud_classic",
	"ibmcloud_vpc",
	"k8s_api",
	"kubeadm",
	"openshift",
	"openstack_nova",
	"oraclecloud",
	"scaleway",
	"system",
	"tencent_cvm",
	"upcloud",
	"vultr",
}

// buildTags returns the full go build -tags value for the distribution: the
// user-provided BuildTags combined with the tags needed to omit resource detectors.
//
// If no detectors are configured under dist.resource.detection.detectors, every
// detector is left in and no detector-related tags are generated. If one or more
// detectors are configured, every other known detector is omitted via a matching
// omit_detector_<name> tag, so that only the configured detectors remain compiled in.
func (d Distribution) buildTags() string {
	tags := d.BuildTags

	configured := make(map[string]struct{}, len(d.Resource.Detection.Detectors))
	for _, detector := range d.Resource.Detection.Detectors {
		for name := range detector {
			configured[name] = struct{}{}
		}
	}
	if len(configured) == 0 {
		return tags
	}

	var omitTags []string
	for _, name := range knownResourceDetectors {
		if _, ok := configured[name]; !ok {
			omitTags = append(omitTags, omitDetectorBuildTag+name)
		}
	}

	if len(omitTags) > 0 {
		if tags != "" {
			tags += ","
		}
		tags += strings.Join(omitTags, ",")
	}
	return tags
}

// Module represents a receiver, exporter, processor or extension for the distribution
type Module struct {
	Name   string `mapstructure:"name,omitempty"`   // if not specified, this is package part of the go mod (last part of the path)
	Import string `mapstructure:"import,omitempty"` // if not specified, this is the path part of the go mods
	GoMod  string `mapstructure:"gomod,omitempty"`  // a gomod-compatible spec for the module
	Path   string `mapstructure:"path,omitempty"`   // an optional path to the local version of this module
}

type retry struct {
	numRetries int
	wait       time.Duration
}

// NewDefaultConfig creates a new config, with default values
func NewDefaultConfig() (*Config, error) {
	log, err := zap.NewDevelopment()
	if err != nil {
		panic(fmt.Sprintf("failed to obtain a logger instance: %v", err))
	}

	outputDir, err := os.MkdirTemp("", "otelcol-distribution")
	if err != nil {
		return nil, err
	}

	return &Config{
		OtelColVersion: DefaultBetaOtelColVersion,
		Logger:         log,
		Distribution: Distribution{
			OutputPath: outputDir,
			Module:     "go.opentelemetry.io/collector/cmd/builder",
		},
		// basic retry if error from go mod command (in case of transient network error).
		// retry 3 times with 5 second spacing interval
		downloadModules: retry{
			numRetries: 3,
			wait:       5 * time.Second,
		},
		ConfmapProviders: []Module{
			{
				GoMod: "go.opentelemetry.io/collector/confmap/provider/envprovider " + DefaultStableOtelColVersion,
			},
			{
				GoMod: "go.opentelemetry.io/collector/confmap/provider/fileprovider " + DefaultStableOtelColVersion,
			},
			{
				GoMod: "go.opentelemetry.io/collector/confmap/provider/httpprovider " + DefaultStableOtelColVersion,
			},
			{
				GoMod: "go.opentelemetry.io/collector/confmap/provider/httpsprovider " + DefaultStableOtelColVersion,
			},
			{
				GoMod: "go.opentelemetry.io/collector/confmap/provider/yamlprovider " + DefaultStableOtelColVersion,
			},
		},
	}, nil
}

// Validate checks whether the current configuration is valid
func (c *Config) Validate() error {
	return multierr.Combine(
		validateModules("extension", c.Extensions),
		validateModules("receiver", c.Receivers),
		validateModules("exporter", c.Exporters),
		validateModules("processor", c.Processors),
		validateModules("connector", c.Connectors),
		validateModules("provider", c.ConfmapProviders),
		validateModules("converter", c.ConfmapConverters),
		validateTelemetry(c),
		validateResourceDetectors(c.Distribution.Resource.Detection.Detectors),
	)
}

// validateResourceDetectors ensures every detector configured under
// dist.resource.detection.detectors is a name buildTags knows how to omit the others for.
func validateResourceDetectors(detectors []ResourceDetector) error {
	known := make(map[string]struct{}, len(knownResourceDetectors))
	for _, name := range knownResourceDetectors {
		known[name] = struct{}{}
	}

	var errs error
	for _, detector := range detectors {
		for name := range detector {
			if _, ok := known[name]; !ok {
				errs = multierr.Append(errs, fmt.Errorf("%s: %w", name, errUnknownResourceDetector))
			}
		}
	}
	return errs
}

// SetGoPath sets go path
func (c *Config) SetGoPath() error {
	if !c.SkipCompilation || !c.SkipGetModules {
		//nolint:gosec // #nosec G204
		if _, err := exec.Command(c.Distribution.Go, "env").CombinedOutput(); err != nil {
			path, err := exec.LookPath("go")
			if err != nil {
				return ErrGoNotFound
			}
			c.Distribution.Go = path
		}
		c.Logger.Info("Using go", zap.String("go-executable", c.Distribution.Go))
	}
	return nil
}

// ParseModules will parse the Modules entries and populate the missing values
func (c *Config) ParseModules() error {
	var err error
	usedNames := make(map[string]int)

	c.Extensions, err = c.parseModules(c.Extensions, usedNames)
	if err != nil {
		return err
	}

	c.Receivers, err = c.parseModules(c.Receivers, usedNames)
	if err != nil {
		return err
	}

	c.Exporters, err = c.parseModules(c.Exporters, usedNames)
	if err != nil {
		return err
	}

	c.Processors, err = c.parseModules(c.Processors, usedNames)
	if err != nil {
		return err
	}

	c.Connectors, err = c.parseModules(c.Connectors, usedNames)
	if err != nil {
		return err
	}

	telemetry, err := c.parseModules([]Module{c.Telemetry}, usedNames)
	if err != nil {
		return err
	}
	c.Telemetry = telemetry[0]

	c.ConfmapProviders, err = c.parseModules(c.ConfmapProviders, usedNames)
	if err != nil {
		return err
	}
	c.ConfmapConverters, err = c.parseModules(c.ConfmapConverters, usedNames)
	if err != nil {
		return err
	}
	return nil
}

func (c *Config) allComponents() []Module {
	return slices.Concat(c.Exporters, c.Receivers, c.Processors, c.Extensions, c.Connectors, []Module{c.Telemetry}, c.ConfmapProviders, c.ConfmapConverters)
}

func validateModules(name string, mods []Module) error {
	for i, mod := range mods {
		if mod.GoMod == "" {
			return fmt.Errorf("%s module at index %v: %w", name, i, errMissingGoMod)
		}
	}
	return nil
}

// validateTelemetry ensures there is a valid telemetry module specified.
// If the field is not set, it is defaulted to otelconftelemetry.
func validateTelemetry(c *Config) error {
	// We cannot set this in createDefaultConfig, since koanf merges maps and we
	// would get a blend of this value and user-provided values. Once
	// otelconftelemetry is its own module (that is, the `Import` field is not
	// set), we can likely move the default to createDefaultConfig.
	if c.Telemetry.Name == "" && c.Telemetry.Import == "" && c.Telemetry.GoMod == "" && c.Telemetry.Path == "" {
		c.Telemetry = Module{
			GoMod:  "go.opentelemetry.io/collector/service " + DefaultBetaOtelColVersion,
			Import: "go.opentelemetry.io/collector/service/telemetry/otelconftelemetry",
		}
	} else if c.Telemetry.GoMod == "" {
		return fmt.Errorf("telemetry module: %w", errMissingGoMod)
	}

	return nil
}

func (c *Config) parseModules(mods []Module, usedNames map[string]int) ([]Module, error) {
	var parsedModules []Module
	for _, mod := range mods {
		if mod.Import == "" {
			mod.Import = strings.Split(mod.GoMod, " ")[0]
		}

		if mod.Name == "" {
			parts := strings.Split(mod.Import, "/")
			mod.Name = parts[len(parts)-1]
		}

		originalModName := mod.Name
		if count, exists := usedNames[mod.Name]; exists {
			var newName string
			for {
				newName = fmt.Sprintf("%s%d", mod.Name, count+1)
				if _, transformedExists := usedNames[newName]; !transformedExists {
					break
				}
				count++
			}
			mod.Name = newName
			usedNames[newName] = 1
		}
		usedNames[originalModName] = 1

		// Check if path is empty, otherwise filepath.Abs replaces it with current path ".".
		if mod.Path != "" {
			var err error
			absPath, err := filepath.Abs(mod.Path)
			if err != nil {
				return mods, fmt.Errorf("failed to resolve absolute path for %s: %w", mod.Path, err)
			}

			if c.Distribution.UseAbsoluteReplacePaths {
				mod.Path = absPath
			} else {
				absOutputPath, err := filepath.Abs(c.Distribution.OutputPath)
				if err != nil {
					return mods, fmt.Errorf("failed to resolve absolute path for output dir %s: %w", c.Distribution.OutputPath, err)
				}
				mod.Path, err = filepath.Rel(absOutputPath, absPath)
				if err != nil {
					return mods, fmt.Errorf("failed to make path relative to output dir: %w", err)
				}
			}
			mod.Path = filepath.ToSlash(mod.Path)

			// Check if the path exists using the absolute path
			if _, err := os.Stat(absPath); os.IsNotExist(err) {
				return mods, fmt.Errorf("filepath does not exist: %s", absPath)
			}
		}

		parsedModules = append(parsedModules, mod)
	}

	return parsedModules, nil
}

// MarshalYAML encodes Config to YAML using mapstructure tags, omitting zero values.
func (c Config) MarshalYAML() (any, error) {
	return structToMap(c), nil
}

// structToMap converts a struct to a map[string]any using mapstructure tags.
// Fields tagged with mapstructure:"-" are skipped.
// Fields tagged with omitempty are omitted when zero.
func structToMap(v any) map[string]any {
	rv := reflect.ValueOf(v)
	rt := rv.Type()
	result := make(map[string]any)

	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		fv := rv.Field(i)

		if !field.IsExported() {
			continue
		}

		tag := field.Tag.Get("mapstructure")
		if tag == "" {
			continue
		}

		parts := strings.SplitN(tag, ",", 2)
		key := parts[0]
		if key == "-" {
			continue
		}
		omitempty := len(parts) == 2 && parts[1] == "omitempty"

		val := encodeValue(fv)
		if omitempty && isEmpty(val) {
			continue
		}

		result[key] = val
	}

	return result
}

// encodeValue recursively encodes a reflect.Value for use in a YAML map.
// Structs are converted via structToMap, slices are encoded element by element,
// pointers are dereferenced (nil pointers become nil), and all other kinds are
// returned as-is.
func encodeValue(rv reflect.Value) any {
	switch rv.Kind() {
	case reflect.Struct:
		return structToMap(rv.Interface())
	case reflect.Pointer:
		if rv.IsNil() {
			return nil
		}
		return encodeValue(rv.Elem())
	case reflect.Slice:
		if rv.IsNil() {
			return nil
		}
		s := make([]any, rv.Len())
		for i := range rv.Len() {
			s[i] = encodeValue(rv.Index(i))
		}
		return s
	default:
		return rv.Interface()
	}
}

func isEmpty(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	//nolint:exhaustive
	switch rv.Kind() {
	case reflect.Map:
		return rv.Len() == 0
	case reflect.Slice:
		return rv.IsNil() || rv.Len() == 0
	default:
		return rv.IsZero()
	}
}
