# Build otel-collector deb package

Build the otel-collector deb package with [fpm](https://github.com/jordansissel/fpm).

To build the deb package, run `make deb-package` from the repo root directory. The deb package will be written to
`bin/otel-collector_<version>_amd64.deb`.

By default, `<version>` will be the latest git tag with `-post` appended, e.g. `1.2.3-post`.  To explicitly set the
version, run `VERSION=X.Y.X make deb-package`.