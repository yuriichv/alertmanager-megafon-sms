#!/bin/sh
DIR=${PWD}
cd $(dirname "$0")  
mkdir ${PWD}/{bin,gopath} -p
export GOBIN=${PWD}/bin
export GOPATH=${PWD}/gopath
go get -u  -ldflags "-X main.sha1ver=`git rev-parse HEAD` -X main.buildTime=$(date +'%d-%m-%YT%T')" github.com/yuriichv/alertmanager-megafon-sms
cd $DIR

