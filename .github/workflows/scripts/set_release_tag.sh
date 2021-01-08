TAG="${GITHUB_REF##*/}"
if [[ $TAG =~ ^v[0-9]+\.[0-9]+\.[0-9]+.* ]]
then
    echo "::set-output name=tag::$TAG"
fi
