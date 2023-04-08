PACKAGES := cmd/winhchk

GO ?= go
export GOOS ?= windows
export GOARCH ?= amd64

ifeq ($(OS),Windows_NT)
	RM = del /Q
	Path = $(subst /,\,$1)
else
	RM = rm -f
	Path = $1
endif

default: clean test build

build: build/winhchk.exe
.PHONY: build

build/winhchk.exe: $(wildcard cmd/winhchk/*.go)
	$(GO) build -ldflags="-s -w" -o build/winhchk.exe ./cmd/winhchk

clean:
	$(RM) $(call Path,build/*)
.PHONY: clean

fmt:
	$(GO) fmt $(addprefix ./,$(PACKAGES))
.PHONY: fmt

get:
	go get ./cmd/winhchk
.PHONY: get

test:
	$(GO) test $(addprefix ./,$(PACKAGES))
.PHONY: test

upgrade:
	$(GO) get -u ./cmd/winhchk
	$(GO) mod tidy
.PHONY: upgrade
