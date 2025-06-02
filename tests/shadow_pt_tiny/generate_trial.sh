#!/bin/bash 

trialdir=$1

mkdir $trialdir

tar -xvf splitpt-trial.tar --directory=$trialdir --strip-components=1
