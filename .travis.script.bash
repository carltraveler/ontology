#!/usr/bin/env bash

VERSION=$(git describe --always --tags --long)

if [ $TRAVIS_OS_NAME != 'windows' ]; then
	env GO111MODULE=on make all
	env GO111MODULE=on go mod vendor
	cd ./wasmtest && bash ./run-wasm-tests.sh && cd ../
	bash ./.travis.check-license.sh
	bash ./.travis.gofmt.sh
	bash ./.travis.gotest.sh
else
	CGO_ENABLED=1 go build  -ldflags "-X github.com/ontio/ontology/common/config.Version=${VERSION}" -o ontology-windows-amd64 main.go
	go build  -ldflags "-X github.com/ontio/ontology/common/config.Version=${VERSION}" -o sigsvr-windows-amd64 sigsvr.go
fi
