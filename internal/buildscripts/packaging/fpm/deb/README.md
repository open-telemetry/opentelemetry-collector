# Build otel-collector deb package

Build the otel-collector deb package with [fpm](https://github.com/jordansissel/fpm).

To build the deb package, run `make deb-package` from the repo root directory. The deb package will be written to
`dist/otel-collector_<version>_<arch>.deb`.

By default, `<arch>` is `amd64` and `<version>` is the latest git tag with `-post` appended, e.g. `1.2.3-post`.
To override these defaults, set the `ARCH` and `VERSION` environment variables, e.g.
`ARCH=arm64 VERSION=4.5.6 make deb-package`.

Run `./internal/buildscripts/packaging/fpm/test.sh PATH_TO_DEB_FILE` to run a basic installation test with the built
package.