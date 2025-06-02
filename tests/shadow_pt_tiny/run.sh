#!/bin/bash

# this should run inside the container

#set -euo pipefail

simdir=$1

#tar -xaf tornet__net-0.01__load-3.2__trial-1__pt-obfs4.tar.xz

SEED=`python3 -c 'import random; print(random.randint(0, 999999999))'`

tornettools simulate \
    --shadow /opt/bin/shadow \
    --args "--parallelism=32 --seed=${SEED} --template-directory=shadow.data.template --model-unblocked-syscall-latency=true --strace-logging-mode=standard" \
    --filename shadow.config.yaml \
    ${simdir}

# parses the results and saves results in the simdir
tornettools parse ${simdir}
# generates the plots of the results
tornettools plot ${simdir} --prefix ${simdir}/pdfs
# goes through all the log files and compresses them
tornettools archive ${simdir}

chmod -R 777 ${simdir} 
