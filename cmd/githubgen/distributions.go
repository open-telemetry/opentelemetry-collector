// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

type distributionsGenerator struct{}

type distOutput struct {
	Name        string              `yaml:"name"`
	URL         string              `yaml:"url"`
	Maintainers []string            `yaml:"maintainers"`
	Components  map[string][]string `yaml:"components"`
}

func (cg distributionsGenerator) generate(data *githubData) error {
	for _, dist := range data.distributions {
		components := map[string][]string{}
		for _, c := range data.components {
			inDistro := false
			for _, componentDistro := range c.Status.Distributions {
				if dist.Name == componentDistro {
					inDistro = true
					break
				}
			}
			if inDistro {
				array, ok := components[c.Status.Class]
				if !ok {
					array = []string{}
				}
				components[c.Status.Class] = append(array, c.Type)
			}
		}
		for _, comps := range components {
			sort.Strings(comps)
		}
		output := distOutput{
			Name:        dist.Name,
			URL:         dist.URL,
			Maintainers: dist.Maintainers,
			Components:  components,
		}
		b, err := yaml.Marshal(output)
		if err != nil {
			return nil
		}
		err = os.WriteFile(filepath.Join("reports", "distributions", fmt.Sprintf("%s.yaml", dist.Name)), b, 0o600)
		if err != nil {
			return nil
		}
	}
	return nil
}
