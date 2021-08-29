files=(
    bin/otelcol_darwin_amd64
    bin/otelcol_darwin_arm64
    bin/otelcol_linux_arm64
    bin/otelcol_linux_amd64
    bin/otelcol_windows_amd64.exe
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
