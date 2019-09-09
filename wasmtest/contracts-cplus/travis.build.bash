#!/usr/bin/env bash
set -ev

oldir=$(pwd)
currentdir=$(dirname $0)
cd $currentdir

#git clone https://github.com/ontio/ontology-wasm-cdt-cpp
#compilerdir="/home/cl/github/ontology-wasm-cdt-cpp/install/bin"
wget https://github.com/ontio/ontology-wasm-cdt-cpp/blob/master/docker/Dockerfile
docker build -t ontowasm .
docker run -v $(pwd):/root/contracts ontowasm '/bin/bash<<<"cd /root/contracts; for f in *.cpp; do ont_cpp $f -lbase58 -lcrypto -lbuiltins -o  ${f%.cpp}.wasm; done"'
#compilerdir="./ontology-wasm-cdt-cpp/install/bin"

#for f in *.cpp
#do
#	$compilerdir/ont_cpp $f -lbase58 -lcrypto -lbuiltins -o  ${f%.cpp}.wasm
#done

mv *.wasm ../testwasmdata/
rm *.wasm.str

cd $oldir
