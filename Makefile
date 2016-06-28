.PHONY: start-etcd stop-etcd test examples all
.DEFAULT_GOAL := all

export ARGS_DOCKER_HOST=localhost
DOCKER_MACHINE_IP=$(shell docker-machine ip default)
ifneq ($(DOCKER_MACHINE_IP),)
	ARGS_DOCKER_HOST=$(DOCKER_MACHINE_IP)
endif

ETCD_DOCKER_IMAGE=quay.io/coreos/etcd:v3.0.0-beta.0

start-etcd:
	@echo Checking Docker Containers
	@if [ $(shell docker ps -a | grep -ci args-etcd) -eq 0 ]; then \
		echo Starting Docker Container args-etcd; \
		docker run -d -v /usr/share/ca-certificates/:/etc/ssl/certs -p 4001:4001 -p 2380:2380 -p 2379:2379 \
		--name args-etcd $(ETCD_DOCKER_IMAGE) \
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

stop-etcd:
	@if [ $(shell docker ps -a | grep -ci args-etcd) -eq 1 ]; then \
		echo Stopping Container args-etcd; \
		docker stop args-etcd > /dev/null; \
	fi

test: start-etcd
	@echo Running Tests
	@go test .

bin/etcd-config-service: examples/etcd-config-service.go
	go build -o bin/etcd-config-service examples/etcd-config-service.go

bin/etcd-config-client: examples/etcd-config-client.go
	go build -o bin/etcd-config-client examples/etcd-config-client.go

bin/etcd-endpoints-service: examples/etcd-endpoints-service.go
	go build -o bin/etcd-endpoints-service examples/etcd-endpoints-service.go

bin/etcd-endpoints-client: examples/etcd-endpoints-client.go
	go build -o bin/etcd-endpoints-client examples/etcd-endpoints-client.go

bin/chicken-http-client: examples/chicken-http-client.go
	go build -o bin/chicken-http-client examples/chicken-http-client.go

bin/demo: examples/demo.go
	go build -o bin/demo examples/demo.go

all: examples

examples: bin/etcd-endpoints-service bin/etcd-endpoints-client bin/etcd-config-service bin/etcd-config-client bin/chicken-http-client bin/demo

clean:
	rm bin/*

