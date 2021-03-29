#!/bin/bash -e
# Copyright Contributors to the Open Cluster Management project

_script_dir=$(dirname "$0")
mkdir -p test/unit/coverage

echo 'mode: atomic' > test/unit/coverage/cover.out
echo '' > test/unit/coverage/cover.tmp
echo -e "${GOPACKAGES// /\\n}" | xargs -n1 -I{} $_script_dir/test-package.sh {} ${GOPACKAGES// /,}

if [ ! -f test/unit/coverage/cover.out ]; then
    echo "Coverage file test/unit/coverage/cover.out does not exist"
    exit 0
fi

COVERAGE=$(go tool cover -func=test/unit/coverage/cover.out | grep "total:" | awk '{ print $3 }' | sed 's/[][()><%]/ /g')
echo "-------------------------------------------------------------------------"
echo "TOTAL COVERAGE IS ${COVERAGE}%"
echo "-------------------------------------------------------------------------"

go tool cover -html=test/unit/coverage/cover.out -o=test/unit/coverage/cover.html
