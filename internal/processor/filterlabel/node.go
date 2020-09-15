// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package filterlabel

import (
	"errors"
	"fmt"
	"regexp"
)

type node interface {
	match(string) bool
	next() []node
}

func newNode(cfg Config) (node, error) {
	nodes, err := newNodes(cfg.next)
	if err != nil {
		return nil, err
	}
	switch cfg.matchType {
	case MatchTypeRegexp:
		return regexpNode{
			r:     regexp.MustCompile(cfg.str),
			nodes: nodes,
		}, nil
	case MatchTypeStrict:
		return strictNode{
			s:     cfg.str,
			nodes: nodes,
		}, nil
	case "":
		return nil, errors.New("matchType cannot be empty")
	default:
		return nil, fmt.Errorf("invalid matchType %q", cfg.matchType)
	}
}

func newNodes(cfgs []Config) ([]node, error) {
	var out []node
	for _, cfg := range cfgs {
		n, err := newNode(cfg)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, nil
}

// strictNode

type strictNode struct {
	s     string
	nodes []node
}

func (n strictNode) next() []node {
	return n.nodes
}

func (n strictNode) match(s string) bool {
	return n.s == s
}

// regexpNode

type regexpNode struct {
	r     *regexp.Regexp
	nodes []node
}

func (n regexpNode) match(s string) bool {
	return n.r.MatchString(s)
}

func (n regexpNode) next() []node {
	return n.nodes
}
