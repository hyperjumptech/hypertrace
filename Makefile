GOPATH=$(shell go env GOPATH)
EXEC_NAME=HTrace
CURRENT_PATH=$(shell pwd)
GO111MODULE=on
IMAGE_REGISTRY=dockerhub
IMAGE_NAMESPACE ?= hyperjump
IMAGE_NAME=hypertrace
COMMIT_ID ?= $(shell git rev-parse --short HEAD)

.PHONY: all test clean build docker

clean:
	rm -f $(IMAGE_NAME).app

build: clean
	GO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o $(EXEC_NAME).app cmd/*.go

docker-build: build
	docker build -t $(IMAGE_NAMESPACE)/$(IMAGE_NAME):$(COMMIT_ID) -f ./.docker/Dockerfile .
	docker tag $(IMAGE_NAMESPACE)/$(IMAGE_NAME):$(COMMIT_ID) $(IMAGE_NAMESPACE)/$(IMAGE_NAME):latest

docker-push: docker-build
	docker push $(IMAGE_NAMESPACE)/$(IMAGE_NAME):$(COMMIT_ID)