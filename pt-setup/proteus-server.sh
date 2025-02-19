#!/bin/bash

export TOR_PT_MANAGED_TRANSPORT_VER=1
export TOR_PT_SERVER_BINDADDR=proteus-0.0.0.0:9090
export TOR_PT_SERVER_TRANSPORTS=proteus
export TOR_PT_SERVER_TRANSPORT_OPTIONS=proteus:psf=shadowsocks.psf
export TOR_PT_ORPORT=127.0.0.1:8080
export TOR_PT_STATE_LOCATION=$(pwd)/server-state/


proteus
