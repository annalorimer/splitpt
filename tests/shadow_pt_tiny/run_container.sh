#!/bin/bash

docker run --rm -it --name one_splitpt_cxn_one_sptbridge --shm-size=1024g --security-opt seccomp=unconfined -v /home/cc/splitpt/tests/shadow_pt_tiny/:/mnt/ shadow_pt_tiny
