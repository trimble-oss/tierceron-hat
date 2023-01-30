GIT ?= git
GO_VARS ?=
GO ?= go
COMMIT := $(shell $(GIT) rev-parse HEAD)
VERSION ?= $(shell $(GIT) describe --tags ${COMMIT} 2> /dev/null || echo "$(COMMIT)")
BUILD_TIME := $(shell LANG=en_US date +"%F_%T_%z")
ROOT := .
LD_FLAGS := -X $(ROOT).Version=$(VERSION) -X $(ROOT).Commit=$(COMMIT) -X $(ROOT).BuildTime=$(BUILD_TIME)
GOBIN ?= ./bin

.PHONY: help clean 

depend:
	go mod tidy

clean:
	rm -f bin

brimcrown: */*.go
	$(GO_VARS) $(GO) build -tags "" -o="$(ROOT)/bin/brim" -ldflags="$(LD_FLAGS)" $(ROOT)/brim/brim.go
	$(GO_VARS) $(GO) build -tags "" -o="$(ROOT)/bin/crown" -ldflags="$(LD_FLAGS)" $(ROOT)/crown/crown.go

cleanmsdk:
	rm mashupsdk/mashupsdk_grpc.pb.go; rm mashupsdk/mashupsdk.pb.go

mashupsdk: */*.proto
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative mashupsdk/mashupsdk.proto
