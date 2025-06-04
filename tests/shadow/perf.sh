#/bin/bash

check_host () {
    local NAME=$1
    echo ${NAME}
    # These are successes, print the download times
    cat shadow.data/hosts/${NAME}/tgen.*.stdout \
        | grep "stream-success" \
        | cut -d' ' -f15 \
        | cut -d',' -f11
    # These are failures, print the error types
    cat shadow.data/hosts/${NAME}/tgen.*.stdout \
        | grep stream-error \
        | cut -d' ' -f11,13 \
        | cut -d',' -f9,13,14 \
        | sed 's/\] \[/,/g' \
        | sed 's/\]//g' \
        | cut -d',' -f1,3,4 --output-delimiter=' '
    echo ""
}

for host in clientgood clientgoodsplit clientbad clientbadsplit
do
    check_host ${host}
done
