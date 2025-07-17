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

brim: */*.go
	$(GO_VARS) $(GO) build -tags "" -o="$(ROOT)/bin/brim" -ldflags="$(LD_FLAGS)" $(ROOT)/brim/brim.go
	chmod 700 $(ROOT)/bin/brim

brimfeather: */*.go
	$(GO_VARS) $(GO) build -tags "" -o="$(ROOT)/bin/brimfeather" -ldflags="$(LD_FLAGS)" $(ROOT)/brimfeather/brimfeather.go
	chmod 700 $(ROOT)/bin/brimfeather

capfull: */*.go
	$(GO_VARS) GOARCH=wasm GOOS=js $(GO) build -tags "" -o="$(ROOT)/bin/capfull.wasm" -ldflags="$(LD_FLAGS)" $(ROOT)/capfull/capfull.go

captip: */*.go
	$(GO_VARS) $(GO) build -tags "" -o="$(ROOT)/bin/tipcap" -ldflags="$(LD_FLAGS)" $(ROOT)/captip/captip.go
	chmod 700 $(ROOT)/bin/tipcap

captiptwo: */*.go
	$(GO_VARS) $(GO) build -tags "" -o="$(ROOT)/bin/tipcaptwo" -ldflags="$(LD_FLAGS)" $(ROOT)/captiptwo/captip.go
	chmod 700 $(ROOT)/bin/tipcaptwo

captap: */*.go
	$(GO_VARS) $(GO) build -tags "" -o="$(ROOT)/bin/captap" -ldflags="$(LD_FLAGS)" $(ROOT)/captap/captap.go
	chmod 700 $(ROOT)/bin/captap

crown: */*.go
	$(GO_VARS) $(GO) build -tags "" -o="$(ROOT)/bin/crown" -ldflags="$(LD_FLAGS)" $(ROOT)/crown/main.go
	chmod 700 $(ROOT)/bin/crown

tiara: */*.go
	$(GO_VARS) $(GO) build -tags "" -o="$(ROOT)/bin/tiara" -ldflags="$(LD_FLAGS)" $(ROOT)/tiara/main.go
	chmod 700 $(ROOT)/bin/tiara

cleangrpc:
	rm cap/cap_grpc.pb.go; rm cap/cap.pb.go

capgrpc: */*.proto
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative cap/cap.proto

all: tiara brimfeather captip captiptwo