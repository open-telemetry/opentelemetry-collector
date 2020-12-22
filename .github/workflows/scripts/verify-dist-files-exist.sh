files=(
    bin/otelcol_darwin_amd64
    bin/otelcol_linux_arm64
    bin/otelcol_linux_amd64
    bin/otelcol_windows_amd64.exe
    dist/otel-collector-*arm64.rpm
    dist/otel-collector_*amd64.deb
    dist/otel-collector-*x86_64.rpm
    dist/otel-collector_*arm64.deb
    dist/otel-collector-*amd64.msi
);
for f in "${files[@]}"
do
    if [[ ! -f $f ]]
    then
        echo "$f does not exist."
        echo "::set-output name=passed::false"
        exit 0 
    fi
done
echo "::set-output name=passed::true"
