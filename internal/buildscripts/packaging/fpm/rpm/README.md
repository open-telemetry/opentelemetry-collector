# Build otel-collector rpm package

Build the otel-collector rpm package with [fpm](https://github.com/jordansissel/fpm).

To build the rpm package, run `make rpm-package` from the repo root directory. The rpm package will be written to
`dist/otel-collector-<version>.<arch>.rpm`.

By default, `<arch>` is `amd64` and `<version>` is the latest git tag with `~post` appended, e.g. `1.2.3~post`.
To override these defaults, set the `ARCH` and `VERSION` environment variables, e.g.
`ARCH=arm64 VERSION=4.5.6 make rpm-package`.

Run `./internal/buildscripts/packaging/fpm/test.sh PATH_TO_RPM_FILE` to run a basic installation test with the built
package.