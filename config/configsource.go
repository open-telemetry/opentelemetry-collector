// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"context"
	"errors"
	"github.com/spf13/viper"
	"go.opentelemetry.io/collector/component"
	"sort"
	"strings"
)

const (
	// ConfigSourcePrefix is used to identified a configuration source invocation.
	ConfigSourcePrefix = "$"
)

// ApplyConfigSourcesParams holds the parameters for injecting data from the
// given configuration sources into a configuration.
type ApplyConfigSourcesParams struct {
	// Set of configuration sources available to inject data into the configuration.
	ConfigSources map[string]component.ConfigSource
	// MaxRecursionDepth limits the maximum number of times that a configuration source
	// can inject another into the config.
	MaxRecursionDepth uint
	// LoadCount tracks the number of the configuration load session.
	LoadCount int
}

// ApplyConfigSources takes a viper object with the configuration to which the
// the configuration sources will be applied and the resulting configuration is
// returned as a separated object.
func ApplyConfigSources(ctx context.Context, v *viper.Viper, params ApplyConfigSourcesParams) (*viper.Viper, error) {

	sessionParams := component.SessionParams{
		LoadCount: params.LoadCount,
	}

	// Notify all sources that we are about to start.
	for _, cfgSrc := range params.ConfigSources {
		if err := cfgSrc.BeginSession(ctx, sessionParams); err != nil {
			return nil, err // TODO: wrap error with more information.
		}
		defer cfgSrc.EndSession(ctx, sessionParams)
	}

	srcCfg := v
	var dstCfg *viper.Viper

	var err error
	done := false
	for i := uint(0); i <= params.MaxRecursionDepth && !done; i++ {
		dstCfg = NewViper()
		done, err = applyConfigSources(ctx, srcCfg, dstCfg, params.ConfigSources)
		if err != nil {
			return nil, err
		}
		srcCfg = dstCfg
	}

	if !done {
		return nil, errors.New("couldn't fully expand the configuration")
	}

	return dstCfg, nil
}

func applyConfigSources(ctx context.Context, srcCfg, dstCfg *viper.Viper, cfgSources map[string]component.ConfigSource) (bool, error) {
	// Expand any item env vars in the config, do it every time so env vars
	// added on previous pass are also handled.
	expandEnvConfig(srcCfg)

	done := true
	appliedTags := make(map[string]struct{})

	// Apply tags from the deepest config sources to the lowest so nested invocations are properly resolved.
	allKeys := srcCfg.AllKeys()
	sort.Slice(allKeys, deepestConfigSourcesFirst(allKeys))

	for _, k := range allKeys {
		dstKey, cfgSrcName, paramsKey := extractCfgSrcInvocation(k)
		if cfgSrcName == "" {
			// Nothing to apply take the key and value as it is.
			dstCfg.Set(k, srcCfg.Get(k))
			continue
		}

		if _, ok := appliedTags[dstKey]; ok {
			// Already applied this key.
			continue
		}

		appliedTags[dstKey] = struct{}{}
		cfgSrc, ok := cfgSources[cfgSrcName]
		if !ok {
			return false, errors.New("unknown config source") // TODO: error similar to other config errors.
		}

		applyParams := srcCfg.Get(paramsKey)
		actualCfg, err := cfgSrc.Apply(ctx, applyParams)
		if err != nil {
			return false, err // TODO: Add config source name to the error.
		}

		// The injection may require further expansion, assume that we are not done yet.
		done = false
		if dstKey != "" {
			dstCfg.Set(dstKey, actualCfg)
			continue
		}

		// This is at the root level, have to inject the top keys one by one.
		rootMap, ok := actualCfg.(map[string]interface{})
		if !ok {
			return false, errors.New("only a map can be injected at the root level") // TODO: better error info.
		}
		for k, v := range rootMap {
			dstCfg.Set(k, v)
		}
	}

	return done, nil
}

func deepestConfigSourcesFirst(keys []string) func(int, int) bool {
	return func(i, j int) bool {
		iLastSrcIdx := strings.LastIndex(keys[i], ConfigSourcePrefix)
		jLastSrcIdx := strings.LastIndex(keys[j], ConfigSourcePrefix)
		if iLastSrcIdx != jLastSrcIdx {
			// At least one of the keys has a config source.
			return iLastSrcIdx > jLastSrcIdx
		}

		// It can be one of the following cases:
		//  1. None has a config source invocation
		//  2. Both have at the same level, ie.: they are siblings in the config.
		// In any case it doesn't matter which one is considered "smaller".
		return true
	}
}

func extractCfgSrcInvocation(k string) (dstKey, cfgSrcName, paramsKey string) {
	firstPrefixIdx := strings.Index(k, ConfigSourcePrefix)
	if firstPrefixIdx == -1 {
		// No config source to be applied.
		return
	}

	// Check for a deeper one.
	tagPrefixIdx := strings.LastIndex(k, ConfigSourcePrefix)
	dstKey = strings.TrimSuffix(k[:tagPrefixIdx], ViperDelimiter)

	cfgSrcFromStart := k[tagPrefixIdx+len(ConfigSourcePrefix):]
	prefixToTagEndLen := strings.Index(cfgSrcFromStart, ViperDelimiter)
	if prefixToTagEndLen > -1 {
		cfgSrcName = cfgSrcFromStart[:prefixToTagEndLen]
	} else {
		cfgSrcName = cfgSrcFromStart
	}

	paramsKey = k[:tagPrefixIdx+len(cfgSrcName)+len(ConfigSourcePrefix)]

	return
}
