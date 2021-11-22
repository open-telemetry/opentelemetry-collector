package defaultconfigprovider

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cast"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configmapprovider"
	"gopkg.in/yaml.v2"
)

const (
	// configSourceNameDelimChar is the char used to terminate the name of config source
	// when it is used to retrieve values to inject in the configuration
	configSourceNameDelimChar = ':'
	// expandPrefixChar is the char used to prefix strings that can be expanded,
	// either environment variables or config sources.
	expandPrefixChar = '$'
	// typeAndNameSeparator is the separator that is used between type and name in type/name
	// composite keys.
	typeAndNameSeparator = '/'
)

// private error types to help with testability
type (
	errUnknownConfigSource struct{ error }
)

type valueSubstitutor struct {
	onChange      func(event *configmapprovider.ChangeEvent)
	retrieved     configmapprovider.RetrievedConfig
	configSources map[config.ComponentID]configmapprovider.BaseProvider
}

func (vp *valueSubstitutor) Get(ctx context.Context) (*config.Map, error) {
	cfgMap, err := vp.retrieved.Get(ctx)
	if err != nil {
		return nil, err
	}

	return vp.substitute(ctx, cfgMap)
}

func (vp *valueSubstitutor) substitute(ctx context.Context, cfgMap *config.Map) (*config.Map, error) {
	for _, k := range cfgMap.AllKeys() {
		val, err := vp.parseConfigValue(ctx, cfgMap.Get(k))
		if err != nil {
			return nil, err
		}
		cfgMap.Set(k, val)
	}

	return cfgMap, nil
}

func (vp *valueSubstitutor) Close(ctx context.Context) error {
	return vp.retrieved.Close(ctx)
}

// parseConfigValue takes the value of a "config node" and process it recursively. The processing consists
// in transforming invocations of config sources and/or environment variables into literal data that can be
// used directly from a `config.Map` object.
func (vp *valueSubstitutor) parseConfigValue(ctx context.Context, value interface{}) (interface{}, error) {
	switch v := value.(type) {
	case string:
		// Only if the value of the node is a string it can contain an env var or config source
		// invocation that requires transformation.
		return vp.parseStringValue(ctx, v)
	case []interface{}:
		// The value is of type []interface{} when an array is used in the configuration, YAML example:
		//
		//  array0:
		//    - elem0
		//    - elem1
		//  array1:
		//    - entry:
		//        str: elem0
		//	  - entry:
		//        str: $tstcfgsrc:elem1
		//
		// Both "array0" and "array1" are going to be leaf config nodes hitting this case.
		nslice := make([]interface{}, 0, len(v))
		for _, vint := range v {
			value, err := vp.parseConfigValue(ctx, vint)
			if err != nil {
				return nil, err
			}
			nslice = append(nslice, value)
		}
		return nslice, nil
	case map[string]interface{}:
		// The value is of type map[string]interface{} when an array in the configuration is populated with map
		// elements. From the case above (for type []interface{}) each element of "array1" is going to hit the
		// the current case block.
		nmap := make(map[interface{}]interface{}, len(v))
		for k, vint := range v {
			value, err := vp.parseConfigValue(ctx, vint)
			if err != nil {
				return nil, err
			}
			nmap[k] = value
		}
		return nmap, nil
	default:
		// All other literals (int, boolean, etc) can't be further expanded so just return them as they are.
		return v, nil
	}
}

// parseStringValue transforms environment variables and config sources, if any are present, on
// the given string in the configuration into an object to be inserted into the resulting configuration.
func (vp *valueSubstitutor) parseStringValue(ctx context.Context, s string) (interface{}, error) {
	// Code based on os.Expand function. All delimiters that are checked against are
	// ASCII so bytes are fine for this operation.
	var buf []byte

	// Using i, j, and w variables to keep correspondence with os.Expand code.
	// i tracks the index in s from which a slice to be appended to buf should start.
	// j tracks the char being currently checked and also the end of the slice to be appended to buf.
	// w tracks the number of characters being consumed after a prefix identifying env vars or config sources.
	i := 0
	for j := 0; j < len(s); j++ {
		// Skip chars until a candidate for expansion is found.
		if s[j] == expandPrefixChar && j+1 < len(s) {
			if buf == nil {
				// Assuming that the length of the string will double after expansion of env vars and config sources.
				buf = make([]byte, 0, 2*len(s))
			}

			// Append everything consumed up to the prefix char (but not including the prefix char) to the result.
			buf = append(buf, s[i:j]...)

			var expandableContent, cfgSrcName string
			w := 0 // number of bytes consumed on this pass

			switch {
			case s[j+1] == expandPrefixChar:
				// Escaping the prefix so $$ becomes a single $ without attempting
				// to treat the string after it as a config source or env var.
				expandableContent = string(expandPrefixChar)
				w = 1 // consumed a single char

			case s[j+1] == '{':
				// Bracketed usage, consume everything until first '}' exactly as os.Expand.
				expandableContent, w = scanToClosingBracket(s[j+1:])
				expandableContent = strings.Trim(expandableContent, " ") // Allow for some spaces.
				delimIndex := strings.Index(expandableContent, string(configSourceNameDelimChar))
				if len(expandableContent) > 1 && delimIndex > -1 {
					// Bracket expandableContent contains ':' treating it as a config source.
					cfgSrcName = expandableContent[:delimIndex]
				}

			default:
				// Non-bracketed usage, ie.: found the prefix char, it can be either a config
				// source or an environment variable.
				var name string
				name, w = getTokenName(s[j+1:])
				expandableContent = name // Assume for now that it is an env var.

				// Peek next char after name, if it is a config source name delimiter treat the remaining of the
				// string as a config source.
				possibleDelimCharIndex := j + w + 1
				if possibleDelimCharIndex < len(s) && s[possibleDelimCharIndex] == configSourceNameDelimChar {
					// This is a config source, since it is not delimited it will consume until end of the string.
					cfgSrcName = name
					expandableContent = s[j+1:]
					w = len(expandableContent) // Set consumed bytes to the length of expandableContent
				}
			}

			// At this point expandableContent contains a string to be expanded, evaluate and expand it.
			switch {
			case cfgSrcName == "":
				// Not a config source, expand as os.ExpandEnv
				buf = osExpandEnv(buf, expandableContent, w)

			default:
				// A config source, retrieve and apply results.
				retrieved, err := vp.retrieveConfigSourceData(ctx, cfgSrcName, expandableContent)
				if err != nil {
					return nil, err
				}

				consumedAll := j+w+1 == len(s)
				if consumedAll && len(buf) == 0 {
					// This is the only expandableContent on the string, config
					// source is free to return interface{} but parse it as YAML
					// if it is a string or byte slice.
					switch value := retrieved.(type) {
					case []byte:
						if err := yaml.Unmarshal(value, &retrieved); err != nil {
							// The byte slice is an invalid YAML keep the original.
							retrieved = value
						}
					case string:
						if err := yaml.Unmarshal([]byte(value), &retrieved); err != nil {
							// The string is an invalid YAML keep it as the original.
							retrieved = value
						}
					}

					if mapIFace, ok := retrieved.(map[interface{}]interface{}); ok {
						// yaml.Unmarshal returns map[interface{}]interface{} but config
						// map uses map[string]interface{}, fix it with a cast.
						retrieved = cast.ToStringMap(mapIFace)
					}

					return retrieved, nil
				}

				// Either there was a prefix already or there are still characters to be processed.
				if retrieved == nil {
					// Since this is going to be concatenated to a string use "" instead of nil,
					// otherwise the string will end up with "<nil>".
					retrieved = ""
				}

				buf = append(buf, fmt.Sprintf("%v", retrieved)...)
			}

			j += w    // move the index of the char being checked (j) by the number of characters consumed (w) on this iteration.
			i = j + 1 // update start index (i) of next slice of bytes to be copied.
		}
	}

	if buf == nil {
		// No changes to original string, just return it.
		return s, nil
	}

	// Return whatever was accumulated on the buffer plus the remaining of the original string.
	return string(buf) + s[i:], nil
}

// retrieveConfigSourceData retrieves data from the specified config source and injects them into
// the configuration. The Manager tracks sessions and watcher objects as needed.
func (vp *valueSubstitutor) retrieveConfigSourceData(ctx context.Context, cfgSrcName string, cfgSrcInvocation string) (interface{}, error) {
	cfgSrcID, err := config.NewComponentIDFromString(cfgSrcName)
	if err != nil {
		return nil, err
	}

	configSrc, ok := vp.configSources[cfgSrcID]
	if !ok {
		return nil, newErrUnknownConfigSource(cfgSrcName)
	}

	valueSrc, ok := configSrc.(configmapprovider.ValueProvider)
	if !ok {
		return nil, fmt.Errorf("config source %q is not a ValueProvider, cannot use with ${%v:...} syntax", cfgSrcName, cfgSrcName)
	}

	cfgSrcName, selector, paramsConfigMap, err := parseCfgSrcInvocation(cfgSrcInvocation)
	if err != nil {
		return nil, err
	}

	// Recursively expand the selector.
	var expandedSelector interface{}
	expandedSelector, err = vp.parseStringValue(ctx, selector)
	if err != nil {
		return nil, fmt.Errorf("failed to process selector for config source %q selector %q: %w", cfgSrcName, selector, err)
	}
	if selector, ok = expandedSelector.(string); !ok {
		return nil, fmt.Errorf("processed selector must be a string instead got a %T %v", expandedSelector, expandedSelector)
	}

	// Recursively resolve/parse any config source on the parameters.
	if paramsConfigMap != nil {
		paramsConfigMap, err = vp.substitute(ctx, paramsConfigMap)
		if err != nil {
			return nil, fmt.Errorf("failed to process parameters for config source %q invocation %q: %w", cfgSrcName, cfgSrcInvocation, err)
		}
	}

	retrieved, err := valueSrc.Retrieve(ctx, vp.onChange, selector, paramsConfigMap)
	if err != nil {
		return nil, fmt.Errorf("config source %q failed to retrieve value: %w", cfgSrcName, err)
	}

	valMap, err := retrieved.Get(ctx)
	if err != nil {
		return nil, err
	}

	return valMap, nil
}

func newErrUnknownConfigSource(cfgSrcName string) error {
	return &errUnknownConfigSource{
		fmt.Errorf(`config source %q not found if this was intended to be an environment variable use "${%s}" instead"`, cfgSrcName, cfgSrcName),
	}
}

// parseCfgSrcInvocation parses the original string in the configuration that has a config source
// retrieve operation and return its "logical components": the config source name, the selector, and
// a config.Map to be used in this invocation of the config source. See Test_parseCfgSrcInvocation
// for some examples of input and output.
// The caller should check for error explicitly since it is possible for the
// other values to have been partially set.
func parseCfgSrcInvocation(s string) (cfgSrcName, selector string, paramsConfigMap *config.Map, err error) {
	parts := strings.SplitN(s, string(configSourceNameDelimChar), 2)
	if len(parts) != 2 {
		err = fmt.Errorf("invalid config source syntax at %q, it must have at least the config source name and a selector", s)
		return
	}
	cfgSrcName = strings.Trim(parts[0], " ")

	// Separate multi-line and single line case.
	afterCfgSrcName := parts[1]
	switch {
	case strings.Contains(afterCfgSrcName, "\n"):
		// Multi-line, until the first \n it is the selector, everything after as YAML.
		parts = strings.SplitN(afterCfgSrcName, "\n", 2)
		selector = strings.Trim(parts[0], " ")

		if len(parts) > 1 && len(parts[1]) > 0 {
			var cp *config.Map
			cp, err = config.NewMapFromBuffer(bytes.NewReader([]byte(parts[1])))
			if err != nil {
				return
			}
			paramsConfigMap = cp
		}

	default:
		// Single line, and parameters as URL query.
		const selectorDelim string = "?"
		parts = strings.SplitN(parts[1], selectorDelim, 2)
		selector = strings.Trim(parts[0], " ")

		if len(parts) == 2 {
			paramsPart := parts[1]
			paramsConfigMap, err = parseParamsAsURLQuery(paramsPart)
			if err != nil {
				err = fmt.Errorf("invalid parameters syntax at %q: %w", s, err)
				return
			}
		}
	}

	return cfgSrcName, selector, paramsConfigMap, err
}

func parseParamsAsURLQuery(s string) (*config.Map, error) {
	values, err := url.ParseQuery(s)
	if err != nil {
		return nil, err
	}

	// Transform single array values in scalars.
	params := make(map[string]interface{})
	for k, v := range values {
		switch len(v) {
		case 0:
			params[k] = nil
		case 1:
			var iface interface{}
			if err = yaml.Unmarshal([]byte(v[0]), &iface); err != nil {
				return nil, err
			}
			params[k] = iface
		default:
			// It is a slice add element by element
			elemSlice := make([]interface{}, 0, len(v))
			for _, elem := range v {
				var iface interface{}
				if err = yaml.Unmarshal([]byte(elem), &iface); err != nil {
					return nil, err
				}
				elemSlice = append(elemSlice, iface)
			}
			params[k] = elemSlice
		}
	}
	return config.NewMapFromStringMap(params), err
}

// osExpandEnv replicate the internal behavior of os.ExpandEnv when handling env
// vars updating the buffer accordingly.
func osExpandEnv(buf []byte, name string, w int) []byte {
	switch {
	case name == "" && w > 0:
		// Encountered invalid syntax; eat the
		// characters.
	case name == "" || name == "$":
		// Valid syntax, but $ was not followed by a
		// name. Leave the dollar character untouched.
		buf = append(buf, expandPrefixChar)
	default:
		buf = append(buf, os.Getenv(name)...)
	}

	return buf
}

// scanToClosingBracket consumes everything until a closing bracket '}' following the
// same logic of function getShellName (os package, env.go) when handling environment
// variables with the "${<env_var>}" syntax. It returns the expression between brackets
// and the number of characters consumed from the original string.
func scanToClosingBracket(s string) (string, int) {
	for i := 1; i < len(s); i++ {
		if s[i] == '}' {
			if i == 1 {
				return "", 2 // Bad syntax; eat "${}"
			}
			return s[1:i], i + 1
		}
	}
	return "", 1 // Bad syntax; eat "${"
}

// getTokenName consumes characters until it has the name of either an environment
// variable or config source. It returns the name of the config source or environment
// variable and the number of characters consumed from the original string.
func getTokenName(s string) (string, int) {
	if len(s) > 0 && isShellSpecialVar(s[0]) {
		// Special shell character, treat it os.Expand function.
		return s[0:1], 1
	}

	var i int
	firstNameSepIdx := -1
	for i = 0; i < len(s); i++ {
		if isAlphaNum(s[i]) {
			// Continue while alphanumeric plus underscore.
			continue
		}

		if s[i] == typeAndNameSeparator && firstNameSepIdx == -1 {
			// If this is the first type name separator store the index and continue.
			firstNameSepIdx = i
			continue
		}

		// It is one of the following cases:
		// 1. End of string
		// 2. Reached a non-alphanumeric character, preceded by at most one
		//    typeAndNameSeparator character.
		break
	}

	if firstNameSepIdx != -1 && (i >= len(s) || s[i] != configSourceNameDelimChar) {
		// Found a second non alpha-numeric character before the end of the string
		// but it is not the config source delimiter. Use the name until the first
		// name delimiter.
		return s[:firstNameSepIdx], firstNameSepIdx
	}

	return s[:i], i
}

// Below are helper functions used by os.Expand, copied without changes from original sources (env.go).

// isShellSpecialVar reports whether the character identifies a special
// shell variable such as $*.
func isShellSpecialVar(c uint8) bool {
	switch c {
	case '*', '#', '$', '@', '!', '?', '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return true
	}
	return false
}

// isAlphaNum reports whether the byte is an ASCII letter, number, or underscore
func isAlphaNum(c uint8) bool {
	return c == '_' || '0' <= c && c <= '9' || 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z'
}
