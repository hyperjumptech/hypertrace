GOPATH=$(shell go env GOPATH)
IMAGE_NAME=OTMock
CURRENT_PATH=$(shell pwd)
GO111MODULE=on

.PHONY: all test clean build docker

clean:
	rm -f $(IMAGE_NAME).app

build:
	GO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o $(IMAGE_NAME).app cmd/*.go


