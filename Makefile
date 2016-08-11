.PHONY: test examples all get-deps
.DEFAULT_GOAL := all

# GO
GOPATH := $(shell go env | grep GOPATH | sed 's/GOPATH="\(.*\)"/\1/')
GLIDE := $(GOPATH)/bin/glide
PATH := $(GOPATH)/bin:$(PATH)
export $(PATH)

bin/chicken-http-client: examples/chicken-http-client.go
	go build -o bin/chicken-http-client examples/chicken-http-client.go

bin/demo: examples/demo.go
	go build -o bin/demo examples/demo.go

all: test examples

examples: bin/chicken-http-client bin/demo

travis-ci: get-deps
	go get -u github.com/mattn/goveralls
	go get -u golang.org/x/tools/cmd/cover
	goveralls -service=travis-ci

$(GLIDE):
	go get -u github.com/Masterminds/glide

get-deps: $(GLIDE)
	$(GLIDE) install

clean:
	rm bin/*

test:
	go test .
