# `ocb init --from-config`: build a manifest from a Collector configuration

`ocb init --from-config <file>` reads an existing Collector configuration,
enumerates the receivers, processors, exporters, extensions, and connectors it
references, and emits a `manifest.yaml` containing the Go modules for those
components, limited to the official `opentelemetry-collector` and
`opentelemetry-collector-contrib` repositories.

Module paths are verified against the Go module proxy before being written, so
no unverified path ever reaches the manifest.

Components that cannot be resolved produce a warning and a commented-out
placeholder line for manual completion.

## Motivation

Going from "I have a working Collector configuration" to "I have a custom
distribution that runs it" today requires hand-translating every component ID
in the configuration into a Go module path, version-pinning each one, and
filling in a `manifest.yaml` from scratch.
That's a rather hard process, and can be a blocker for folks who may want to
start using their own distribution, but have one or multiple collector(s)
already in production.

`ocb init` already scaffolds a new distribution, but it emits a fixed
two-component manifest (`otlpreceiver` and `otlpexporter`) regardless of the
user's actual configuration.

## Scope and non-goals

**In scope:** the five component classes (receivers, processors, exporters,
extensions, connectors), resolved against official core and
`opentelemetry-collector-contrib` modules only.

**Non-goals:**

- *Never emit an unverified module path.* This is the central commitment of the
	proposal. An unresolvable component is reported, not guessed.
- *`ocb` does not gain a component inventory to maintain.* No embedded table of
	component types in this repository, and no obligation for contrib to publish
	one.
- Resolving confmap providers and converters (already defaulted by `ocb`), or
	`service::telemetry`.
- Resolving third-party or vendor components. Their presence in the
	configuration is expected and produces a warning.
- Validating that the configuration is semantically correct. That is the
	Collector's job (or possibly an `ocb lint/validate` command later on).

## Explanation

```
ocb init --path ./mycol --from-config ./collector-config.yaml
```

`ocb` reads the given configuration file and resolves each component ID to a Go
module.
The generated `manifest.yaml` contains a `gomod` entry for every resolved
component.
The supplied configuration file is copied to `<path>/config.yaml` (instead of
rendering the static `config.yaml.tmpl`), so the generated `Makefile`'s `run`
target (which passes `--config ../../config.yaml`) actually runs the
configuration the manifest was built for.

Unresolved components produce:

1. A warning on stderr:
   ```
   ocb: warning: no Go module found for extension "myvendor"; add its gomod to manifest.yaml manually
   ocb: resolved 7 of 8 components referenced by ./collector-config.yaml
   ```
2. A commented-out placeholder block appended to `manifest.yaml` after the YAML body:
   ```yaml
   # The following components are referenced by ./collector-config.yaml but could
   # not be matched to a Go module in the opentelemetry-collector or
   # opentelemetry-collector-contrib repositories. Fill in the module path and
   # version for each one and uncomment it, or remove it from your Collector
   # configuration.
   #
   # extensions:
   #     - gomod: <module path> <version> # myvendor
   ```

Without `--from-config`, `ocb init` behaviour is unchanged.
Re-running `ocb init --from-config` against an existing distribution directory
follows the same file-overwrite policy as `ocb init` without the flag.

### Example

Given this configuration:

```yaml
receivers:
  otlp:
  hostmetrics:
  prometheus_simple:
processors:
  batch:
  k8sattributes:
exporters:
  debug:
  otlphttp:
extensions:
  file_storage:
  myvendor:
service:
  extensions: [file_storage]
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [debug]
```

`ocb init --path ./mycol --from-config ./config.yaml` produces (among other
files) a
`manifest.yaml` like:

```yaml
dist:
    description: Custom OpenTelemetry Collector
    name: mycol
    output_path: ./build/collector
exporters:
    - gomod: go.opentelemetry.io/collector/exporter/debugexporter v0.161.0
    - gomod: go.opentelemetry.io/collector/exporter/otlphttpexporter v0.161.0
extensions:
    - gomod: github.com/open-telemetry/opentelemetry-collector-contrib/extension/storage/filestorage v0.161.0
processors:
    - gomod: go.opentelemetry.io/collector/processor/batchprocessor v0.161.0
    - gomod: github.com/open-telemetry/opentelemetry-collector-contrib/processor/k8sattributesprocessor v0.161.0
receivers:
    - gomod: go.opentelemetry.io/collector/receiver/otlpreceiver v0.161.0
    - gomod: github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver v0.161.0
    - gomod: github.com/open-telemetry/opentelemetry-collector-contrib/receiver/simpleprometheusreceiver v0.161.0

# The following components are referenced by ./config.yaml but could
# not be matched to a Go module in the opentelemetry-collector or
# opentelemetry-collector-contrib repositories. Fill in the module path and
# version for each one and uncomment it, or remove it from your Collector
# configuration.
#
# extensions:
#     - gomod: <module path> <version> # myvendor
```

## Internal details

### Step 1: Enumerate component IDs

`ocb` reads the five top-level sections of the configuration (`receivers`,
`processors`, `exporters`, `extensions`, `connectors`) using
`go.yaml.in/yaml/v3`, already a direct dependency of `cmd/builder`.
No environment variable expansion is performed: expanding `${env:FOO}` in map
keys would require importing `confmap`, which `cmd/builder` deliberately avoids
(see [cmd/builder module constraints](#cmdbuildertoolmodule-constraints)).

A component ID is `type` or `type/name`; the type is the substring before the
first `/`.
IDs in the same class deduplicate (`otlp` and `otlp/2` yield one component).
Null or absent sections produce no IDs.
Non-mapping sections (e.g. a section supplied entirely via an environment
variable) produce an error.
Unlike an unresolved component, where the type string is known and a
commented-out placeholder can be emitted, the component list itself is unknown
and cannot be approximated, so graceful degradation is not possible.

### Step 2: Generate candidate module paths

For each component `(class, type)`, `ocb` generates a list of candidate module
paths in priority order.
Let `t` be the type with all underscores removed, `segs` be the type split on
`_`:

| # | Candidate suffix | Condition |
|---|---|---|
| 1 | `<class>/<t><class>` | always |
| 2 | `<class>/<t>` | always |
| 3 | `<class>/<t>auth<class>` | extensions only |
| 4 | `<class>/<segs[-1]>/<join(segs[:-1])><segs[-1]><class>` | `len(segs) > 1` |
| 5 | `<class>/<segs[-1]>/<join(segs[:-1])><segs[-1]>` | `len(segs) > 1` |
| 6 | `<class>/<reversed(segs) joined><class>` | `len(segs) > 1` |

Each suffix is tried under the core prefix (`go.opentelemetry.io/collector/`)
before the contrib prefix
(`github.com/open-telemetry/opentelemetry-collector-contrib/`), in that order.
Duplicates within a component's candidate list are removed.

As a worked example: for `file_storage` (class `extension`), `t =
"filestorage"` and `segs = ["file", "storage"]`. Patterns #1–3 generate paths
directly under `extension/` (`filestorageextension`, `filestorage`,
`filestorageauthextension`), none of which exist on the proxy.
Pattern #4 generates `extension/storage/filestorageextension` (also absent).
Pattern #5 generates `extension/storage/filestorage`, which exists and is the
correct module.

A small hardcoded exceptions map overrides pattern generation for three types
whose module paths are not reachable by any pattern:

| class | type | module |
|---|---|---|
| extension | `asapclient` | contrib `extension/asapauthextension` |
| receiver | `podman_stats` | contrib `receiver/podmanreceiver` |
| exporter | `otlp_grpc` | core `exporter/otlpexporter` |

Exception paths are still verified before being emitted; a future contrib
rename degrades safely to "unresolved" rather than to a bogus entry.

Every generated path is validated with `golang.org/x/mod/module.CheckPath`
before being sent to the proxy.
This rejects malformed paths produced by adversarial type strings, such as a
trailing underscore yielding an empty path element.

### Step 3: Reject non-component modules

Some candidate paths produced by the patterns resolve to real Go modules that
are *not* components; they have no factory function.
Emitting such a path produces a manifest that `go mod tidy` accepts but `ocb
build` (at the `components.go` generation step) rejects with an opaque compile
error.
This is worse than an unresolved warning.

The affected paths are discoverable at design time and fall into a small number
of structural categories.
A candidate path is rejected and the search continues if:

1. It contains `/internal/` or `/examples/`.
2. Its final path element ends with `test` or `helper`.
3. Its final path element is `x` + `<class>`, or starts with `x` and ends with `<class>` or
   `helper` (e.g. `xreceiver`, `xexporterhelper`).
4. It is in the explicit denylist:
   - `github.com/open-telemetry/opentelemetry-collector-contrib/extension/storage`
   - `github.com/open-telemetry/opentelemetry-collector-contrib/extension/observer`
   - `github.com/open-telemetry/opentelemetry-collector-contrib/extension/encoding`
   - `github.com/open-telemetry/opentelemetry-collector-contrib/extension/opampcustommessages`
   - `go.opentelemetry.io/collector/extension/extensionauth`
   - `go.opentelemetry.io/collector/extension/extensioncapabilities`
   - `go.opentelemetry.io/collector/extension/extensionmiddleware`

Filter #4 is exact-path matching, not prefix matching: a candidate like
`extension/storage/filestorage` is not rejected by the denylist entry for
`extension/storage`.

This filter has zero false rejections against the full set of 291 current
component modules (250 contrib + 14 core + 27 deprecated-type aliases), and
rejects 30 of the 33 non-component modules in the same directory trees.
The 3 remaining are unreachable by construction: they are modules whose paths
contain three or more segments below the class directory, which patterns #4 and
#5 cannot produce (those patterns generate at most two segments below the class
directory for any underscore-delimited type string).

### Step 4: Verify candidates against the module proxy

Candidates are verified using `go list -m -e -mod=mod -json <mod>@latest ...`.
The `-e` flag causes per-module failures to be reported in an `Error` field
rather than terminating the process.
The `@latest` query is used rather than the pinned version constant because the
`prepare-release-prs` commit bumps `DefaultBetaOtelColVersion` before the
corresponding tag is pushed, making a pinned-version query fail during the
release window.
Output is a stream of concatenated JSON objects (no enclosing array), decoded
with `json.Decoder.Decode` in a loop.

A resolved module must satisfy all of:
- `Error == ""`
- `semver.IsValid(Version)`, which rejects pseudo-versions (a pre-release contrib module with no tag
  would otherwise be emitted as a commit SHA, surprising users)
- `!module.IsPseudoVersion(Version)`

**Batching.** Each `go list` execution covers all still-unresolved components
at one priority level (one (pattern, prefix) pair).
A miss at the contrib prefix is not merely a 404: Go falls through to `direct`
mode and runs `git ls-remote` against the contrib repository, costing ~0.3 s
per miss.
Level-by-level batching minimises this by issuing only the highest-priority
candidate for each unresolved component in each round.
Since pattern #1 alone resolves 256 of 291 components, the common case finishes
in at most two executions (one for core candidates, one for contrib candidates
at level 1).
Longer queries are chunked to stay within the Windows command-line limit.

**Detecting unavailable module lookups.** The first batch includes one sentinel
module (`go.opentelemetry.io/collector/receiver/otlpreceiver@latest`).
If the sentinel fails, module lookups are unavailable (rather than the
components being absent), and `ocb` exits with an actionable error without
writing any files:

```
failed to verify component modules: module lookup disabled by GOPROXY=off
ocb init --from-config needs access to the Go module proxy; run `ocb init` without
--from-config for a default manifest.
```

**Network access is a stated requirement** of `--from-config`. The module cache
(`$GOMODCACHE`) serves repeat runs after the first.
`ocb init` already runs `go mod tidy`, so a network dependency is not new in
kind.

### Step 5: Select and emit

Among the candidates that passed verification, the highest-priority one (lowest
pattern number, core before contrib within a pattern) is selected.
The zero-collision property makes "first verified candidate wins" safe: no
higher-priority candidate in the 291-component test set is itself a real
component module other than the correct one.

Versions emitted:

- Core hits: the existing `DefaultBetaOtelColVersion` constant (e.g. `v0.161.0`).
- Contrib hits: the version of
	`github.com/open-telemetry/opentelemetry-collector-contrib` resolved at
	`DefaultBetaOtelColVersion`; this root module is real and tagged in lockstep.
	During the brief skew window when the contrib tag lags the core tag, the
	fallback is the `@latest`-resolved version returned by Step 4; that resolved
	version (not `DefaultBetaOtelColVersion`) is what is written to the manifest
	for those contrib entries.

This ensures all contrib entries in the manifest are at the same version as
each other and, in the common case, at the version matching the core entries.

### cmd/builder module constraints

`cmd/builder` carries zero dependencies on any
`go.opentelemetry.io/collector/*` module and no `replace` directives.
This is intentional: `ocb` is released independently of the collector, and
adding collector dependencies would require `replace` directives that conflict
with that model. The proposal respects this constraint:

- Configuration parsing uses `go.yaml.in/yaml/v3` (direct dep).
- Module path validation uses `golang.org/x/mod/module.CheckPath` (direct dep).
- Version handling uses `golang.org/x/mod/semver` (direct dep).
- Component ID syntax (`type` or `type/name`, cut on the first `/`) is replicated locally (~5
  lines) rather than imported from `go.opentelemetry.io/collector/component`.
- No new dependencies on collector repositories are added.

### Testability

Module verification sits behind an injectable function type (`ListFunc`) so
that unit tests never shell out to the Go toolchain or hit the network.
Only the integration test (`cmd/builder/test/test.sh`) exercises real module
resolution, and its `--from-config` case uses a core-only fixture to remain
within the local `replaces:` injected by the script.

## Validation

The analysis below is based on scanning all `metadata.yaml` files in the
`opentelemetry-collector` and `opentelemetry-collector-contrib` repositories at
`DefaultBetaOtelColVersion v0.161.0`, covering 291 component modules (250
contrib + 14 core + 27 deprecated-type aliases).

### Why a naming convention alone is insufficient

The obvious convention `<class>/<type minus underscores><class>` fails for a
significant tail:

| type | class | actual module suffix |
|---|---|---|
| `otlp_grpc` | exporter | `exporter/otlpexporter` (not `otlpgrpcexporter`) |
| `file_storage` | extension | `extension/storage/filestorage` (nested) |
| `redis_storage` | extension | `extension/storage/redisstorageextension` (nested, with suffix) |
| `k8s_observer` | extension | `extension/observer/k8sobserver` (nested) |
| `json_log_encoding` | extension | `extension/encoding/jsonlogencodingextension` (nested, with suffix) |
| `oauth2client` | extension | `extension/oauth2clientauthextension` (auth infix) |
| `prometheus_simple` | receiver | `receiver/simpleprometheusreceiver` (reversed) |
| `receiver_creator` | receiver | `receiver/receivercreator` (no class suffix) |
| `podman_stats` | receiver | `receiver/podmanreceiver` (word drop) |

Convention alone emits unverified wrong module paths for these components.
Verification makes any individual miss safe (it becomes a warning), but
systematic misses for common components would render the feature unhelpful.

### Coverage of the candidate set

Pattern usage against all 291 real component modules (simulation, not live
queries):

| pattern | components resolved | cumulative |
|---|---|---|
| #1 | 256 (87.9%) | 87.9% |
| #2 | 7 (2.4%) | 90.3% |
| #3 | 2 (0.7%) | 91.0% |
| #4 | 14 (4.8%) | 95.8% |
| #5 | 8 (2.7%) | 98.6% |
| #6 | 1 (0.3%) | 98.9% |
| exceptions | 3 (1.0%) | 100% |

**Zero ordering collisions** among the 291 component modules: no
higher-priority candidate for any component is itself a real component module
other than the correct one. "First verified candidate wins" is safe.

(Pattern #7 from an earlier draft, `<class>/<reversed segs>` without the class
suffix, was measured against all 291 components and resolved exactly zero; it
is excluded from the implementation.)

### The denylist

Of the 33 non-component modules under the five class directories, patterns #1–6 are capable of
generating 4 contrib paths that exist on the proxy: `extension/{storage,observer,encoding,
opampcustommessages}`, reachable via pattern #2 from the corresponding type strings. The three
core API modules in the denylist (`extensionauth`, `extensioncapabilities`, `extensionmiddleware`)
are reachable via pattern #2 as well.
The reject filter (rules 1–4 above) catches all 7 without rejecting any of the
291 real component modules.

### Live verification sample

```
$ go list -m -e -mod=mod -json \
    go.opentelemetry.io/collector/receiver/otlpreceiver@latest \
    github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver@latest \
    github.com/open-telemetry/opentelemetry-collector-contrib/receiver/totallybogusreceiver@latest

{"Path":"go.opentelemetry.io/collector/receiver/otlpreceiver","Version":"v0.161.0"}
{"Path":"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver","Version":"v0.161.0"}
{"Path":"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/totallybogusreceiver","Error":{"Err":"...receiver/totallybogusreceiver@latest: no matching versions for query \"latest\""}}
```

The exit code is 0 in all cases; failures are per-object.

### Deprecated type aliases

All 27 `deprecated_type` alias entries across core and contrib resolve
correctly through the candidate patterns.
For example, a config using `exporters: otlp:` yields the type `otlp`, whose
pattern-#1 candidate `exporter/otlpexporter` exists in core, which is the
correct module, even though the component's current canonical type is
`otlp_grpc`.

## Trade-offs and mitigations

**Network dependency.** Module verification requires access to the Go module
proxy.
`ocb init` already requires network access for `go mod tidy`; the additional
dependency is in kind rather than category. The module cache (`$GOMODCACHE`)
serves repeat runs after the first warm-up.

**Latency.** The dominant cost is misses, not hits.
A contrib-prefix miss causes Go to fall through to `direct` and run `git
ls-remote` against the contrib repository (~0.3 s).
Level-wise batching (one exec per priority level over still-unresolved
components) limits this.
With an 87.9% hit rate at level 1 and the remaining ~21% split across 5 more
levels, a typical config of 10 components with mostly-common types completes in
3–5 executions.

**False positives from non-component modules.** The reject filter (rules 1–4)
eliminates the known cases.
An unreachable gap would cause `ocb build` to fail at `components.go`
generation with a compile error.
The reject filter's explicit denylist needs to be updated if a new
non-component module matching rules 1–3 is added under the class directories;
the structural rules (1–3) handle future `integrationtest`-style paths
automatically.

**Contrib renames.** If a contrib component moves to a different module path,
the old path produces a 404, the component becomes "unresolved", and the user
gets a warning and a placeholder.
Degradation is safe by construction.

**`ocb` becoming contrib-aware.** This proposal introduces a contrib module
prefix constant into `ocb`.
The core repository has so far not depended on contrib in this way.

**Third-party type shadows.** A vendor component whose type happens to match a
core or contrib type (e.g. an in-house extension named `k8s_observer`) will
resolve to the contrib module for that type.
Mitigated by unconditionally printing the resolved `type → module` mapping for
every component to stderr during `init`, so the user can spot incorrect
mappings before running the build:

```
ocb: resolved receiver otlp → go.opentelemetry.io/collector/receiver/otlpreceiver v0.161.0
ocb: resolved extension k8s_observer → github.com/open-telemetry/opentelemetry-collector-contrib/extension/observer/k8sobserver v0.161.0
```

**Relative paths in the copied `config.yaml`.** The source config may reference
paths relative to its original location (e.g. `file:./certs/ca.pem`).
After copying to `<path>/config.yaml`, those references will break unless they
also resolve correctly from the new working directory.
Users should use absolute paths for any file references in configs intended for
use with `--from-config`.

**Single `--from-config`.** The flag accepts one file.
Supporting multiple files would require defining merge semantics for the five
sections and a policy for which file is copied to `config.yaml`.
This is not in scope at the moment, but may be handled later.

## Open questions

- **Is adding the contrib module prefix to `ocb` acceptable?** This is the most
	consequential commitment in this proposal. If it is not, the feature's scope
	may need to be limited to core components only.

- **Contrib version pinning strategy.** The proposal pins contrib modules to the
	version of the `github.com/open-telemetry/opentelemetry-collector-contrib`
	root module resolved at `DefaultBetaOtelColVersion`, falling back to the
	`@latest`-resolved version during the brief release skew window (see Step 5).
	Does the community agree with this approach, or prefer always using `@latest`
	for contrib (simpler to implement, but not guaranteed to match core module
	versions in the same manifest)?

- **Unresolved component handling.** Should an unresolved component be a hard
	error (guaranteeing the manifest builds the whole config) or a warn-and-skip
	(allowing partial manifests for configs that mix in third-party components)?

- **`config.yaml` overwrite.** Should `--from-config` copy the configuration to
	`config.yaml` (making the generated `make run` work with the given config) or
	leave the static template (avoiding any implicit overwrite)? The proposal
	says yes to copying; this could be a source of surprising behaviour if the
	user later edits `config.yaml` independently.

## Alternatives considered

**Embedded generated component table (~262 entries).** A `components.yaml` file
in `cmd/builder`, regenerated by a `make` target that scans `metadata.yaml`
files from both repos, would work offline and be exact.
It was rejected because it creates recurring maintainer burden in this
repository for another repository's inventory, goes stale silently (new contrib
components are invisible until someone reruns the generator).
This alternative is worth revisiting if a contrib-published machine- readable
inventory becomes available.

**Convention only, no verification.** The 12 patterns plus exceptions cover
100% of the current component set.
Without verification, the feature would work on any config composed entirely of
known types.
It was rejected because (a) silent misses for unknown types would be worse than
the current state (the user has no indicator something was skipped), and (b)
the non-component module false-positive risk means some inputs produce
build-time compile errors, which are harder to diagnose than warnings.

**`otelcol components` output.** Consuming the `components` subcommand output
of an existing collector binary would give an exact mapping. It was rejected
because it requires the user to already have a distribution they are trying to
create.

**Importing `confmap`/`component`.** These packages parse component IDs
exactly.
They were rejected because they would add collector dependencies and `replace`
directives to `cmd/builder`, which is intentionally dependency-free.

**A contrib-published machine-readable inventory.** The best long-term answer
for correctness and maintenance: contrib publishes a versioned `(class, type,
module)` mapping alongside each release, and `ocb` fetches it.
This is listed as a future possibility rather than a blocker.

## Future possibilities

- Contrib (or a shared build tool) publishes a machine-readable component
	inventory; `ocb` uses it as the primary source, falling back to pattern
	generation.
- An `ocb lint` command that validates the generated manifest.
