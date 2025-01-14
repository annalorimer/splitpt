export TOR_PT_MANAGED_TRANSPORT_VER=1
export TOR_PT_SERVER_BINDADDR=webtunnel-0.0.0.0:9090 
export TOR_PT_SERVER_TRANSPORTS=webtunnel
export TOR_PT_ORPORT=127.0.0.1:8080
export TOR_PT_STATE_LOCATION=server-state/
export TOR_PT_SERVER_TRANSPORT_OPTIONS=webtunnel:url=http://127.0.0.1:9090

#lyrebird -enableLogging -logLevel DEBUG
webtunnel 
