#!/usr/bin/env bash
set -ev

oldir=$(pwd)
currentdir=$(dirname $0)
cd $currentdir

git clone https://github.com/carltraveler/ontology-wasm-cdt-cpp
git checkout compile
cd ontology-wasm-cdt-cpp && bash compiler_install.bash && cd ../
compilerdir="./ontology-wasm-cdt-cpp/install/bin"

for f in $(ls *.cpp)
do
	$compilerdir/ont_cpp $f -lbase58 -lbuiltins -o  ${f%.cpp}.wasm
done

rm -rf ontology-wasm-cdt-cpp
mv *.wasm ../testwasmdata/
rm *.wasm.str
cp  *.avm ../testwasmdata/

cd $oldir
