#!/bin/sh
DIR=${PWD}
cd $(dirname "$0")  
mkdir ${PWD}/{bin,gopath} -p
export GOBIN=${PWD}/bin
export GOPATH=${PWD}/gopath
go get -u github.com/yuriichv/alertmanager-megafon-sms
cd $DIR

