import sys
def getport():
    with open('splitpt-client.1000.stdout', 'r') as fin:
        for line in fin:
            if line.startswith('CMETHOD splitpt socks5 127.0.0.1:'):
                return line.strip().split(' ')[3].split(':')[1]
    return '0'
def getcert(path):
    with open(path,'r') as fin:
        return fin.read().split()[-2]

with open('torrc','r') as fin:
    data = fin.read().replace('SOCKS5LISTENPORT', getport())
with open('torrc','w') as fout:
    fout.write(data)
with open('splitpt-client.toml', 'w') as spttoml:
    data1=spttoml.read().replace('OBFS4CERT1', getcert('../splitpt-obfs4bridge1/obfs4_bridgeline.txt')).replace('OBFS4CERT2', getcert('../splitpt-obfs4bridge2/obfs4_bridgeline.txt'))
