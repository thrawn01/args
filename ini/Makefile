.PHONY: start-containers stop-containers test examples all glide-deps
.DEFAULT_GOAL := all

# GO
GOPATH := $(shell go env | grep GOPATH | sed 's/GOPATH="\(.*\)"/\1/')
GLIDE := $(GOPATH)/bin/glide
PATH := $(GOPATH)/bin:$(PATH)
export $(PATH)

export ARGS_DOCKER_HOST=localhost
DOCKER_MACHINE_IP=$(shell docker-machine ip default 2> /dev/null)
ifneq ($(DOCKER_MACHINE_IP),)
	ARGS_DOCKER_HOST=$(DOCKER_MACHINE_IP)
endif

ETCD_DOCKER_IMAGE=quay.io/coreos/etcd:latest

start-containers:
	@echo Checking Docker Containers
	@if [ $(shell docker ps -a | grep -ci args-etcd) -eq 0 ]; then \
		echo Starting Docker Container args-etcd; \
		docker run -d -v /usr/share/ca-certificates/:/etc/ssl/certs -p 4001:4001 -p 2380:2380 -p 2379:2379 \
		--name args-etcd $(ETCD_DOCKER_IMAGE) /usr/local/bin/etcd \
		--name etcd0 \
		--advertise-client-urls http://${ARGS_DOCKER_HOST}:2379,http://${ARGS_DOCKER_HOST}:4001 \
		--listen-client-urls http://0.0.0.0:2379,http://0.0.0.0:4001 \
		--initial-advertise-peer-urls http://${ARGS_DOCKER_HOST}:2380 \
		--listen-peer-urls http://0.0.0.0:2380 \
		--initial-cluster-token etcd-cluster-1 \
		--initial-cluster etcd0=http://${ARGS_DOCKER_HOST}:2380 \
		--initial-cluster-state new; \
	elif [ $(shell docker ps | grep -ci args-etcd) -eq 0 ]; then \
		echo restarting args-etcd; \
		docker start args-etcd > /dev/null; \
	fi

stop-containers:
	@if [ $(shell docker ps -a | grep -ci args-etcd) -eq 1 ]; then \
		echo Stopping Container args-etcd; \
		docker stop args-etcd > /dev/null; \
	fi

test: start-containers
	@echo Running Tests
	export ETCDCTL_ENDPOINTS=${ARGS_DOCKER_HOST}:2379
	export ETCDCTL_API=3
	go test . -v


all: examples

examples:
	go install $(shell go list ./... | grep -v vendor)

travis-ci: glide-deps start-containers
	go get -u github.com/mattn/goveralls
	go get -u golang.org/x/tools/cmd/cover
	go get -u golang.org/x/text/secure/bidirule
	goveralls -service=travis-ci

$(GLIDE):
	go get -u github.com/Masterminds/glide

go-deps:
	go get golang.org/x/net/context
	go get github.com/onsi/ginkgo
	go get github.com/onsi/gomega
	go get github.com/pborman/uuid

glide-deps: $(GLIDE) go-deps
	$(GLIDE) install

