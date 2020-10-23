#!/usr/bin/make -f

UNAMES := $(shell uname -s)
VERSION := $(shell echo $(shell git describe --tags) | sed 's/^v//')
COMMIT := $(shell git log -1 --format='%H')
CMD_NAME := fx

export GO111MODULE=on
export GOPROXY=https://goproxy.cn,direct

.PHONY: build build-win build-linux go.mod install format docker-web docker-gobuilder

build-win:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -v -ldflags  -o build/$(CMD_NAME).exe ./cmd/$(CMD_NAME)

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -ldflags  -o build/$(CMD_NAME) ./cmd/$(CMD_NAME)

build:
	go build -v -ldflags  -o build/fx ./cmd/fx
	go build -v -ldflags  -o build/prom ./cmd/prom
	go build -v -ldflags  -o build/random ./cmd/random
	go build -v -ldflags  -o build/fxtx ./cmd/fxtx

go.mod:
	@go mod tidy
	@go mod download
	@go mod verify

install:
	@go install -ldflags  -v ./cmd/$(CMD_NAME)

docker-prom:
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o build/prom ./cmd/prom
	@docker rmi -f functionx/fx-prometheus:latest
	@docker build --no-cache -f ./cmd/prom/Dockerfile -t functionx/fx-prometheus:latest .

format:
	@find . -name '*.go' -type f -not -path "*.git*" | xargs gofmt -w -s
