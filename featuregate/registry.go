// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package featuregate // import "go.opentelemetry.io/collector/featuregate"

import (
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/hashicorp/go-version"
)

var (
	globalRegistry = NewRegistry()

	// idRegexp is used to validate the ID of a Gate.
	// IDs' characters must be alphanumeric, dots, or dashes.
	idRegexp = regexp.MustCompile(`^[0-9a-zA-Z\.\-]+$`)
)

// GlobalRegistry returns the global Registry.
func GlobalRegistry() *Registry {
	return globalRegistry
}

type Registry struct {
	gates sync.Map
}

// NewRegistry returns a new empty Registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// RegisterOption allows to configure additional information about a Gate during registration.
type RegisterOption interface {
	apply(g *Gate) error
}

type registerOptionFunc func(g *Gate) error

func (ro registerOptionFunc) apply(g *Gate) error {
	return ro(g)
}

// WithRegisterDescription adds description for the Gate.
func WithRegisterDescription(description string) RegisterOption {
	return registerOptionFunc(func(g *Gate) error {
		g.description = description
		return nil
	})
}

// WithRegisterReferenceURL adds a URL that has all the contextual information about the Gate.
func WithRegisterReferenceURL(referenceURL string) RegisterOption {
	return registerOptionFunc(func(g *Gate) error {
		if _, err := url.Parse(referenceURL); err != nil {
			return fmt.Errorf("WithRegisterReferenceURL: invalid reference URL %q: %w", referenceURL, err)
		}

		g.referenceURL = referenceURL
		return nil
	})
}

func validateVersion(v string) error {
	if _, err := version.NewVersion(v); err != nil {
		return err
	}
	return nil
}

// WithRegisterFromVersion is used to set the Gate "FromVersion".
// The "FromVersion" contains the Collector release when a feature is introduced.
func WithRegisterFromVersion(fromVersion string) RegisterOption {
	return registerOptionFunc(func(g *Gate) error {
		if err := validateVersion(fromVersion); err != nil {
			return fmt.Errorf("WithRegisterFromVersion: %w", err)
		}

		g.fromVersion = fromVersion
		return nil
	})
}

// WithRegisterToVersion is used to set the Gate "ToVersion".
// The "ToVersion", if not empty, contains the last Collector release in which you can still use a feature gate.
// If the feature stage is either "Deprecated" or "Stable", the "ToVersion" is the Collector release when the feature is removed.
func WithRegisterToVersion(toVersion string) RegisterOption {
	return registerOptionFunc(func(g *Gate) error {
		if err := validateVersion(toVersion); err != nil {
			return fmt.Errorf("WithRegisterToVersion: %w", err)
		}

		g.toVersion = toVersion
		return nil
	})
}

// MustRegister like Register but panics if an invalid ID or gate options are provided.
func (r *Registry) MustRegister(id string, stage Stage, opts ...RegisterOption) *Gate {
	g, err := r.Register(id, stage, opts...)
	if err != nil {
		panic(err)
	}
	return g
}

func validateID(id string) error {
	if id == "" {
		return fmt.Errorf("empty ID")
	}

	if !idRegexp.MatchString(id) {
		return fmt.Errorf("invalid character(s) in ID")
	}
	return nil
}

// Register a Gate and return it. The returned Gate can be used to check if is enabled or not.
func (r *Registry) Register(id string, stage Stage, opts ...RegisterOption) (*Gate, error) {
	if err := validateID(id); err != nil {
		return nil, fmt.Errorf("invalid ID %q: %w", id, err)
	}

	g := &Gate{
		id:    id,
		stage: stage,
	}
	for _, opt := range opts {
		err := opt.apply(g)
		if err != nil {
			return nil, fmt.Errorf("failed to apply option: %w", err)
		}
	}
	switch g.stage {
	case StageAlpha, StageDeprecated:
		g.enabled = &atomic.Bool{}
	case StageBeta, StageStable:
		enabled := &atomic.Bool{}
		enabled.Store(true)
		g.enabled = enabled
	default:
		return nil, fmt.Errorf("unknown stage value %q for gate %q", stage, id)
	}
	if (g.stage == StageStable || g.stage == StageDeprecated) && g.toVersion == "" {
		return nil, fmt.Errorf("no removal version set for %v gate %q", g.stage.String(), id)
	}

	if g.fromVersion != "" && g.toVersion != "" {
		from, err := version.NewVersion(g.fromVersion)
		if err != nil {
			return nil, fmt.Errorf("invalid fromVersion %q: %w", g.fromVersion, err)
		}
		to, err := version.NewVersion(g.toVersion)
		if err != nil {
			return nil, fmt.Errorf("invalid toVersion %q: %w", g.toVersion, err)
		}
		if to.LessThan(from) {
			return nil, fmt.Errorf("toVersion %q is before fromVersion %q", g.toVersion, g.fromVersion)
		}
	}

	if _, loaded := r.gates.LoadOrStore(id, g); loaded {
		return nil, fmt.Errorf("attempted to add pre-existing gate %q", id)
	}
	return g, nil
}

// Set the enabled valued for a Gate identified by the given id.
func (r *Registry) Set(id string, enabled bool) error {
	v, ok := r.gates.Load(id)
	if !ok {
		validGates := []string{}
		r.VisitAll(func(g *Gate) {
			validGates = append(validGates, g.ID())
		})
		return fmt.Errorf("no such feature gate %q. valid gates: %v", id, validGates)
	}
	g := v.(*Gate)

	switch g.stage {
	case StageStable:
		if !enabled {
			return fmt.Errorf("feature gate %q is stable, can not be disabled", id)
		}
		fmt.Printf("Feature gate %q is stable and already enabled. It will be removed in version %v and continued use of the gate after version %v will result in an error.\n", id, g.toVersion, g.toVersion)
	case StageDeprecated:
		if enabled {
			return fmt.Errorf("feature gate %q is deprecated, can not be enabled", id)
		}
		fmt.Printf("Feature gate %q is deprecated and already disabled. It will be removed in version %v and continued use of the gate after version %v will result in an error.\n", id, g.toVersion, g.toVersion)
	default:
		g.enabled.Store(enabled)
	}
	return nil
}

// VisitAll visits all the gates in lexicographical order, calling fn for each.
func (r *Registry) VisitAll(fn func(*Gate)) {
	var gates []*Gate
	r.gates.Range(func(key, value any) bool {
		gates = append(gates, value.(*Gate))
		return true
	})
	sort.Slice(gates, func(i, j int) bool {
		return gates[i].ID() < gates[j].ID()
	})
	for i := range gates {
		fn(gates[i])
	}
}
