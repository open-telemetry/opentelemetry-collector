TESTS="$(make -s testbed-list-loadtest | xargs echo|sed 's/ /|/g')"
TESTS=(${TESTS//|/ })
MATRIX="{\"include\":["
curr=""
for i in "${!TESTS[@]}"; do
if (( i > 0 && i % 2 == 0 )); then
    curr+="|${TESTS[$i]}"
else
    if [ -n "$curr" ] && (( i>1 )); then
    MATRIX+=",{\"test\":\"$curr\"}"
    elif [ -n "$curr" ]; then
    MATRIX+="{\"test\":\"$curr\"}"
    fi
    curr="${TESTS[$i]}"
fi
done
MATRIX+="]}"
echo "::set-output name=matrix::$MATRIX"
