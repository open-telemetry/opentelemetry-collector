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

package configsource

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/viper"

	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/internal/configsource/component"
)

const (
	// ConfigSourcePrefix is used to identify a configuration source invocation.
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
}

// ApplyConfigSources takes a viper object with the configuration to which the
// the configuration sources will be applied and the resulting configuration is
// returned as a separated object.
func ApplyConfigSources(ctx context.Context, v *viper.Viper, params ApplyConfigSourcesParams) (*viper.Viper, error) {

	// Notify all sources that we are about to start.
	for cfgSrcName, cfgSrc := range params.ConfigSources {
		if err := cfgSrc.BeginSession(ctx); err != nil {
			return nil, &cfgSrcError{
				msg:  fmt.Sprintf("config source %q begin session error: %v", cfgSrcName, err),
				code: errCfgSrcBeginSession,
			}
		}

		// The scope of usage of the cfgSrc is the whole function so it is fine to defer
		// the EndSession call to the function exit.
		defer cfgSrc.EndSession(ctx)
	}

	srcCfg := v
	var dstCfg *viper.Viper

	var err error
	done := false
	for i := -1; i <= int(params.MaxRecursionDepth) && !done; i++ {
		dstCfg = config.NewViper()
		done, err = applyConfigSources(ctx, srcCfg, dstCfg, params.ConfigSources)
		if err != nil {
			return nil, err
		}
		srcCfg = dstCfg
	}

	if !done {
		return nil, &cfgSrcError{
			msg:  fmt.Sprintf("config source recursion chain is deeper than the allowed maximum of %d", params.MaxRecursionDepth),
			code: errCfgSrcChainTooLong,
		}
	}

	return dstCfg, nil
}

// cfgSrcError private error type used to accurate identify the type of error in tests.
type cfgSrcError struct {
	msg  string          // human readable error message.
	code cfgSrcErrorCode // internal error code.
}

func (e *cfgSrcError) Error() string {
	return e.msg
}

// These are errors that can be returned by ApplyConfigSources function.
// Note that error codes are not part public API, they are for unit testing only.
type cfgSrcErrorCode int

const (
	_ cfgSrcErrorCode = iota // skip 0, start errors codes from 1.
	errCfgSrcChainTooLong
	errOnlyMapAtRootLevel
	errCfgSrcBeginSession
	errCfgSrcNotFound
	errCfgSrcApply
	errNestedCfgSrc
)

func applyConfigSources(ctx context.Context, srcCfg, dstCfg *viper.Viper, cfgSources map[string]component.ConfigSource) (bool, error) {
	// Expand any item env vars in the config, do it every time so env vars
	// added on previous pass are also handled.
	expandEnvConfig(srcCfg)

	done := true
	appliedTags := make(map[string]struct{})

	// It is possible to have config sources injection depending on other config sources:
	//
	// $lower_cfgsrc:
	//   a: "just an example"
	//   b:
	//     $deeper_cfgsrc:
	//       c: true
	//
	// By injecting the deepest ones first they can be resolved in the proper order.
	// See function deepestConfigSourcesFirst for more info.
	//
	// TODO: Bug when lower injection happening after deeper but without the inject values.
	allKeys := srcCfg.AllKeys()
	sort.Slice(allKeys, deepestConfigSourcesFirst(allKeys))

	// Inspect all key from original configuration and set the proper value on the destination config.
	for _, k := range allKeys {
		dstKey, cfgSrcName, paramsKey := extractCfgSrcInvocation(k)

		// Nested injection is not supported. That is the following is not supported:
		//
		// $lower:
		//   lower_param0: false
		//   lower_param1:
		//     $deeper:
		//       key: deeperkey
		//
		if _, parentCfgSrcName, _ := extractCfgSrcInvocation(dstKey); parentCfgSrcName != "" {
			return false, &cfgSrcError{
				msg:  fmt.Sprintf("nested config source usage at %q this is not supported", paramsKey),
				code: errNestedCfgSrc,
			}
		}

		if cfgSrcName == "" {
			// Nothing to apply take the key and value as it is.
			dstCfg.Set(k, srcCfg.Get(k))
			continue
		}

		if _, ok := appliedTags[dstKey]; ok {
			// Already applied this key. If the config source has multiple parameters
			// there will be multiple keys for the same application.
			continue
		}

		appliedTags[dstKey] = struct{}{}
		cfgSrc, ok := cfgSources[cfgSrcName]
		if !ok {
			return false, &cfgSrcError{
				msg:  fmt.Sprintf("config source %q not found", cfgSrcName),
				code: errCfgSrcNotFound,
			}
		}

		applyParams := srcCfg.Get(paramsKey)
		actualCfg, err := cfgSrc.Apply(ctx, applyParams)
		if err != nil {
			return false, &cfgSrcError{
				msg:  fmt.Sprintf("error applying config source %q: %v", cfgSrcName, err),
				code: errCfgSrcApply,
			}
		}

		// The injection may require further expansion, assume that we are not done yet.
		// This is pessimistic, alternatively we could explore the injected configuration
		// to check if it injected other configuration sources or not.
		done = false
		if dstKey != "" {
			dstCfg.Set(dstKey, actualCfg)
			continue
		}

		// This is at the root level, have to inject the top keys one by one.
		rootMap, ok := actualCfg.(map[string]interface{})
		if !ok {
			return false, &cfgSrcError{
				msg:  "only a map can be injected at the root level",
				code: errOnlyMapAtRootLevel,
			}
		}
		for k, v := range rootMap {
			dstCfg.Set(k, v)
		}
	}

	return done, nil
}

// deepestConfigSourcesFirst function returns a "less" function to be used
// with slice.Sort to ensure that the "deepest" config source invocation appears
// first in the sorted slice. The configuration:
//
//   $lower:
//     a: "just an example"
//     b:
//       $deeper:
//         c: true
//
// will have two keys:
//
//   $lower::a
//   $lower::b::$deeper::c
//
// by using deepestConfigSourcesFirst the sorted slice will have "$lower::b::$deeper::c"
// before "$lower::a".
func deepestConfigSourcesFirst(keys []string) func(int, int) bool {
	return func(i, j int) bool {
		iBranch := strings.Split(keys[i], config.ViperDelimiter)
		jBranch := strings.Split(keys[j], config.ViperDelimiter)

		iLastSrcIdx := lastIndexConfigSource(iBranch)
		jLastSrcIdx := lastIndexConfigSource(jBranch)

		if iLastSrcIdx != jLastSrcIdx {
			// Both have at least one of the keys has a config source.
			// Return the deepest one.
			return iLastSrcIdx > jLastSrcIdx
		}

		// It can be one of the following cases:
		//  1. None has a config source invocation
		//  2. Only one has a config source
		// Return the deepest "branch".
		return len(iBranch) > len(jBranch)
	}
}

func lastIndexConfigSource(keyNodes []string) int {
	lastIndexConfigSource := -1
	for i, node := range keyNodes {
		if strings.HasPrefix(node, ConfigSourcePrefix) {
			lastIndexConfigSource = i
		}
	}
	return lastIndexConfigSource
}

// extractCfgSrcInvocation breaks down a key from the configuration if it contains the ConfigSourcePrefix.
// If the key contains the prefix, the return values are as follows:
//
// - dstKey: the key into which the result applying the config source will be injected.
// - cfgSrcName: the name of the config source to be applied.
// - paramsKey: the key of the parameters to be passed on the call to the config source Apply method.
//
// In case the prefix is not present on the key all returned strings have their default value.
func extractCfgSrcInvocation(k string) (dstKey, cfgSrcName, paramsKey string) {
	// Check for the deepest config source prefix.
	tagPrefixIdx := strings.LastIndex(k, ConfigSourcePrefix)
	if tagPrefixIdx == -1 {
		// No config source to be applied.
		return
	}

	dstKey = strings.TrimSuffix(k[:tagPrefixIdx], config.ViperDelimiter)

	cfgSrcFromStart := k[tagPrefixIdx+len(ConfigSourcePrefix):]
	prefixToTagEndLen := strings.Index(cfgSrcFromStart, config.ViperDelimiter)
	if prefixToTagEndLen > -1 {
		cfgSrcName = cfgSrcFromStart[:prefixToTagEndLen]
	} else {
		cfgSrcName = cfgSrcFromStart
	}

	paramsKey = k[:tagPrefixIdx+len(cfgSrcName)+len(ConfigSourcePrefix)]

	return
}

// Copied from config package to avoid exposing as public API.
// TODO: Add local tests covering the code below.

// expandEnvConfig creates a new viper config with expanded values for all the values (simple, list or map value).
// It does not expand the keys.
func expandEnvConfig(v *viper.Viper) {
	for _, k := range v.AllKeys() {
		v.Set(k, expandStringValues(v.Get(k)))
	}
}

func expandStringValues(value interface{}) interface{} {
	switch v := value.(type) {
	default:
		return v
	case string:
		return expandEnv(v)
	case []interface{}:
		nslice := make([]interface{}, 0, len(v))
		for _, vint := range v {
			nslice = append(nslice, expandStringValues(vint))
		}
		return nslice
	case map[interface{}]interface{}:
		nmap := make(map[interface{}]interface{}, len(v))
		for k, vint := range v {
			nmap[k] = expandStringValues(vint)
		}
		return nmap
	}
}

func expandEnv(s string) string {
	return os.Expand(s, func(str string) string {
		// This allows escaping environment variable substitution via $$, e.g.
		// - $FOO will be substituted with env var FOO
		// - $$FOO will be replaced with $FOO
		// - $$$FOO will be replaced with $ + substituted env var FOO
		if str == "$" {
			return "$"
		}
		return os.Getenv(str)
	})
}
