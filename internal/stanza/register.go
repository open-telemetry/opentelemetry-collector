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

package stanza

import (
	// Register parsers and transformers for stanza-based log receivers
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/parser/json"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/parser/regex"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/parser/severity"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/parser/time"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/parser/trace"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/transformer/metadata"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/transformer/restructure"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/transformer/router"
)
