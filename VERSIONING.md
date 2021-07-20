# Versioning

This document describes the versioning policy for this repository. This policy
is designed so that the following goal can be achieved:

**Users are provided a codebase of value that is stable and secure.**

## Policy

* Versioning of this project will be idiomatic of a Go project using [Go
  modules](https://golang.org/ref/mod#versions).
    * [Semantic import
      versioning](https://github.com/golang/go/wiki/Modules#semantic-import-versioning)
      will be used.
        * Versions will comply with [semver 2.0](https://semver.org/spec/v2.0.0.html).
        * If a module is version `v2` or higher, the major version of the module
          must be included as a `/vN` at the end of the module paths used in
          `go.mod` files (e.g., `module go.opentelemetry.io/collector/v2`, `require
          go.opentelemetry.io/collector/v2 v2.0.1`) and in the package import path
          (e.g., `import "go.opentelemetry.io/collector/v2/component"`). This includes the
          paths used in `go get` commands (e.g., `go get
          go.opentelemetry.io/collector/v2@v2.0.1`.  Note there is both a `/v2` and a
          `@v2.0.1` in that example. One way to think about it is that the module
          name now includes the `/v2`, so include `/v2` whenever you are using the
          module name).
        * If a module is version `v0` or `v1`, do not include the major version in
          either the module path or the import path.
    * A single module should exist, rooted at the top level of this repository,
      that contains all packages provided for use outside this repository.
    * Additional modules may be created in this repository to provide for
      isolation of build-time tools or other commands. Such modules should be
      versioned in sync with the `go.opentelemetry.io/collector` module.
    * Experimental modules still under active development will be versioned with a major
      version of `v0` to imply the stability guarantee defined by
      [semver](https://semver.org/spec/v2.0.0.html#spec-item-4).

      > Major version zero (0.y.z) is for initial development. Anything MAY
      > change at any time. The public API SHOULD NOT be considered stable.

    * Configuration structures should be considered part of the public API and backward
      compatibility maintained through any changes made to configuration structures.
         * Because configuration structures are typically instantiated through unmarshalling
           a serialized representation of the structure, and not through structure literals,
           additive changes to the set of exported fields in a configuration structure are
           not considered to break backward compatibility.
         * Struct tags used to configure serialization mechanisms (`yaml:`, `mapstructure:`, etc)
           are to be considered part of the structure definition and must maintain compatibility
           to the same extent as the structure.
* Versioning of the associated [contrib
  repository](https://github.com/open-telemetry/opentelemetry-collector-contrib) of
  this project will be idiomatic of a Go project using [Go
  modules](https://golang.org/ref/mod#versions).
    * [Semantic import
      versioning](https://github.com/golang/go/wiki/Modules#semantic-import-versioning)
      will be used.
        * Versions will comply with [semver 2.0](https://semver.org/spec/v2.0.0.html).
        * If a module is version `v2` or higher, the
          major version of the module must be included as a `/vN` at the end of the
          module paths used in `go.mod` files (e.g., `module
          github.com/open-telemetry/opentelemetry-collector-contrib/processor/k8sprocessor/v2`, `require
          github.com/open-telemetry/opentelemetry-collector-contrib/processor/k8sprocessor/v2 v2.0.1`) and in the
          package import path (e.g., `import
          "github.com/open-telemetry/opentelemetry-collector-contrib/processor/k8sprocessor/v2"`). This includes
          the paths used in `go get` commands (e.g., `go get
          github.com/open-telemetry/opentelemetry-collector-contrib/processor/k8sprocessor/v2@v2.0.1`.  Note there
          is both a `/v2` and a `@v2.0.1` in that example. One way to think about
          it is that the module name now includes the `/v2`, so include `/v2`
          whenever you are using the module name).
        * If a module is version `v0` or `v1`, do not include the major version
          in either the module path or the import path.
    * Configuration structures should be considered part of the public API and backward
      compatibility maintained through any changes made to configuration structures.
        * Because configuration structures are typically instantiated through unmarshalling
          a serialized representation of the structure, and not through structure literals,
          additive changes to the set of exported fields in a configuration structure are
          not considered to break backward compatibility.
    * Modules will be used to encapsulate receivers, processor, exporters,
      extensions, and any other independent sets of related components.
        * Experimental modules still under active development will be versioned with a major
          version of `v0` to imply the stability guarantee defined by
          [semver](https://semver.org/spec/v2.0.0.html#spec-item-4).

          > Major version zero (0.y.z) is for initial development. Anything MAY
          > change at any time. The public API SHOULD NOT be considered stable.

        * Experimental modules will start their versioning at `v0.0.0` and will
          increment their minor version when backwards incompatible changes are
          released and increment their patch version when backwards compatible
          changes are released.
        * Mature modules for which we guarantee a stable public API will
          be versioned with a major version of `v1` or greater.
        * All stable contrib modules of the same major version with this project
          will use the same entire version.
            * Stable modules may be released with an incremented minor or patch
              version even though that module's code has not been changed. Instead
              the only change that will have been included is to have updated that
              modules dependency on this project's stable APIs.
    * Contrib modules will be kept up to date with this project's releases.
* GitHub releases will be made for all releases.
* Go modules will be made available at Go package mirrors.
