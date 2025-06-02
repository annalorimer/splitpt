# This script replaces the cert placeholders in the splitpt-client.toml with the certs from the splitpt bridges bridgelines
import os

def getcert(path):
    with open(path,'r') as fin:
        return fin.read().split()[-2]

print("Fixup-ing certs")
print(os.getcwd())

with open('splitpt-client.toml', 'r') as spttoml:
    data1=spttoml.read().replace('OBFS4CERT1', getcert('../splitpt-obfs4bridge1/obfs4_bridgeline.txt')).replace('OBFS4CERT2', getcert('../splitpt-obfs4bridge2/obfs4_bridgeline.txt'))
with open('splitpt-client.toml', 'w') as spttomlout:
    spttomlout.write(data1)

print("Succesfully fixup-ed certs")
