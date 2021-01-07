#!/usr/bin/env bash
set -ev

oldir=$(pwd)
currentdir=$(dirname $0)
cd $currentdir

rm -f ../testwasmdata/*
mkdir -p ../testwasmdata/
cp  *.avm ../testwasmdata/

cd $oldir
cd ../
go run wasm-test.go
