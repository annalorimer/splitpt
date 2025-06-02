# This script updates the torrc with the correct socks ports from the splitpt stdout

import sys
def getport():
    with open('splitpt-client.1001.stdout', 'r') as fin:
        for line in fin:
            if line.startswith('CMETHOD splitpt socks5 127.0.0.1:'):
                return line.strip().split(' ')[3].split(':')[1]
    return '0'

print("Fixup-ing SOCKS ports")
with open('torrc','r') as fin:
    data = fin.read().replace('SOCKS5LISTENPORT', getport())
with open('torrc','w') as fout:
    fout.write(data)
print("Fixup-ed SOCKS ports")
