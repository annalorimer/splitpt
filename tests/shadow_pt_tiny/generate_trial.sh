#!/bin/bash 

trialdir=$1

mkdir $trialdir

tar -xvf tornet__net-0.01__load-3.2__trial-1__pt-obfs4.tar.xz --directory=$trialdir --strip-components=1
