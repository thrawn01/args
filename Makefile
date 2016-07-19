.PHONY: test examples all get-deps glide
.DEFAULT_GOAL := all

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

glide:
	@which glide > /dev/null ; if [ $$? -eq 1 ] ; then curl https://glide.sh/get | sh ; fi

get-deps: glide
	glide install

clean:
	rm bin/*

test:
	go test .
