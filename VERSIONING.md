# Versioning and stability

The OpenTelemetry Collector SIG produces several artifacts for [a variety of audiences](CONTRIBUTING.md#target-audiences). This document describes the versioning and support policy for these artifacts. These policies are designed so that the following goal can be achieved:

**Users are provided software artifacts of value that are stable and secure.**

The policies are divided depending on the artifact's target audience. While an artifact is supported, [critical bugs](docs/release.md#bugfix-release-criteria) and security vulnerabilities MUST be addressed. The main criteria for the length of support for an artifact is how easy it is for an artifact's target audience to adapt to disruptive changes.

These policies reflect the current consensus of the OpenTelemetry Collector SIG. They are subject to change as the project evolves.

## Software artifacts for end users

Software artifacts intended for [end users](CONTRIBUTING.md#end-users) of the OpenTelemetry Collector include 
- Binary distributions of the OpenTelemetry Collector.
- Go modules that expose Collector components, such as receivers, processors, connectors, extensions and exporters. 

These artifacts are versioned according to the [semantic versioning v2.0.0](https://semver.org/) specification.

### General considerations

Binary distributions produced by the Collector SIG contain components and features with varying [levels of stability](README.md#stability-levels). We abide by the following principles to relate the Collector's version to the stability of its components and features:

* The Collector's core framework behavior MUST be stable in order for a Collector distribution to be v1.0.0 or higher.
* Users can easily understand when they are opting in to use a component or feature that is not stable.
    * The Collector MUST be configurable so that unstable components or features can be excluded ensuring that a fully stable configuration is possible.
    * The Collector's telemetry (e.g. Collector logs) MUST provide the ability to identify usage of unstable components or features.

### Long-term support after v1

The OpenTelemetry Collector SIG provides long-term support for stable binary distributions of the OpenTelemetry Collector and its components. The following policies apply to long-term support for any major version starting on v1:

* A binary distribution of the OpenTelemetry Collector MUST be supported for a minimum of **one year** after the release of the next major version of said distribution.
* Components MUST be supported for a minimum of **6 months** after the release of the next major version of said component or after the component has been marked as deprecated. If a component has been deprecated for 6 months it MAY be removed from a binary distribution of the OpenTelemetry Collector. This does not imply a major version change in the Collector distribution.

## Go modules

Go modules are intended to be used by [component developers](CONTRIBUTING.md#component-developers) and [Collector library users](CONTRIBUTING.md#collector-library-users) of the OpenTelemetry Collector

Unless otherwise specified, the following public API expectations apply to all modules in opentelemetry-collector and opentelemetry-collector-contrib.
As a general rule, stability guarantees of modules versioned as `v1` or higher are aligned with [Go 1 compatibility promise](https://go.dev/doc/go1compat).

### General Go API considerations

OpenTelemetry authors reserve the right to introduce API changes breaking compatibility between minor versions in the following scenarios:
* **Struct literals.** It may be necessary to add new fields to exported structs in the API. Code that uses unkeyed
  struct literals (such as pkg.T{3, "x"}) to create values of these types would fail to compile after such a change.
  However, code that uses keyed literals (pkg.T{A: 3, B: "x"}) will continue to compile. We therefore recommend 
  using OpenTelemetry collector structs with the keyed literals only.
* **Methods.** As with struct fields, it may be necessary to add methods to types. Under some circumstances,
  such as when the type is embedded in a struct along with another type, the addition of the new method may 
  break the struct by creating a conflict with an existing method of the other embedded type. We cannot protect 
  against this rare case and do not guarantee compatibility in such scenarios.
* **Dot imports.** If a program imports a package using `import .`, additional names defined in the imported package
  in future releases may conflict with other names defined in the program. We do not recommend the use of
  `import .` with OpenTelemetry Collector modules.

Unless otherwise specified in the documentation, the following may change in any way between minor versions:
* **String representation**. The `String` or `Error` method of any struct is intended to be human-readable and may
 change its output in any way.
* **Go version compatibility**. Removing support for an unsupported Go version is not considered a breaking change.
* **OS version compatibility**. Removing support for an unsupported OS version is not considered a breaking change. Upgrading or downgrading OS version support per the [platform support](docs/platform-support.md) document is not considered a breaking change.
* **Protocol compatibility**. Changing the default minimum version of a supported protocol (e.g. TLS) or dropping support for protocols when there are security concerns is not considered a breaking change.
* **Dependency updates**. Updating dependencies is not considered a breaking change except when their types are part of the
public API or the update may change the behavior of applications in an incompatible way.
* **Underlying type for interfaces**. If a struct exported as an interface has an experimental
method, this method may change or be removed in a minor version. The method will be published in an
optional interface under an experimental module to signal it is experimental.

### Configuration structures

Configuration structures are part of the public API and backwards
compatibility should be maintained through any changes made to configuration structures.

Unless otherwise specified in the documentation, the following may change in any way between minor versions:
* **Adding new fields to configuration structures**. Because configuration structures are typically instantiated through 
unmarshalling a serialized representation of the structure, and not through structure literals, additive changes to 
the set of exported fields in a configuration structure are not considered to break backward compatibility.
* **Relaxing validation rules**. An invalid configuration struct as defined by its `Validate` method return value
may become valid after a change to the validation rules.

The following are explicitly considered to be breaking changes:
* **Modifying struct tags related to serialization**. Struct tags used to configure serialization mechanisms (`yaml:`, 
`mapstructure:`, etc) are part of the structure definition and must maintain compatibility to the same extent as the 
structure. However, changes are allowed when tag modifications produce a
functionally-equivalent result when serializing or deserializing the structure.
For example, adding a tag to a field so it will not be emitted during serialization
if it has a default value would not alter its value if the serialized representation
were again deserialized, so such a change would be permitted.
* **Making validation rules more strict**. A valid configuration struct as defined by its `Validate` method return value
must continue to be valid after a change to the validation rules, except when the configuration struct would cause an error
on its intended usage (e.g. when calling a method or when passed to any method or function in any module under opentelemetry-collector).

### Module versioning and schema

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
        * Semantic convention packages will contain a complete version identifier in their
          import path to enable concurrent use of multiple convention versions in a single
          application. This identifies the version of the specification used to generate
          the package and is not related to the version of the module containing the package.
    * A single module should exist, rooted at the top level of this repository,
      that contains all packages provided for use outside this repository.
    * Additional modules may be created in this repository to provide for
      isolation of build-time tools, other commands or independent libraries. Such modules should be
      versioned in sync with the `go.opentelemetry.io/collector` module.
    * Experimental modules still under active development will be versioned with a major
      version of `v0` to imply the stability guarantee defined by
      [semver](https://semver.org/spec/v2.0.0.html#spec-item-4).

      > Major version zero (0.y.z) is for initial development. Anything MAY
      > change at any time. The public API SHOULD NOT be considered stable.
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
    * Modules will be used to encapsulate receivers, processors, exporters,
      extensions, connectors and any other independent sets of related components.
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

### Long-term support after v1

The OpenTelemetry Collector SIG provides long-term support for stable Go modules. Support for modules depend on the module's [target audiences](CONTRIBUTING.md#target-audiences). The following policies apply to long-term support for any major version starting on v1:

- Modules intended for **component developers** MUST be supported for a minimum of **1 year** after the release of the next major version of said module or after the module has been marked as deprecated.
- Modules intended for **Collector library users** MUST be supported for a minimum of **6 months** after the release of the next major version of said module or after the module has been marked as deprecated.
