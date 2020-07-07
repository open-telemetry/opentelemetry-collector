# Build otel-collector rpm package

Build the otel-collector rpm package with [fpm](https://github.com/jordansissel/fpm).

To build the rpm package, run `make rpm-package` from the repo root directory. The rpm package will be written to
`bin/otel-collector-<version>.x86_64.rpm`.

By default, `<version>` will be the latest git tag with `~post` appended, e.g. `1.2.3~post`.  To explicitly set the
version, run `VERSION=X.Y.X make rpm-package`.