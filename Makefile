export GOPROXY=https://goproxy.io
export GO111MODULE=on

HOMEDIR := $(shell pwd)

all: mod build

mod:
	go mod tidy -v

build:
	bash $(HOMEDIR)/build.sh

initdb:
	jutkey-server initDatabase
start:
	jutkey-server start

startup: initdb start
