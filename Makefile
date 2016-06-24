.PHONY: start-docker stop-docker test
.DEFAULT_GOAL := all

export ARGS_DOCKER_HOST=localhost
DOCKER_MACHINE_IP=$(shell docker-machine ip default)
ifneq ($(DOCKER_MACHINE_IP),)
	ARGS_DOCKER_HOST=$(DOCKER_MACHINE_IP)
endif

start-docker:
	@echo Checking Docker Containers
	@if [ $(shell docker ps -a | grep -ci args-etcd) -eq 0 ]; then \
		echo Starting Docker Container args-etcd; \
		docker run -d -v /usr/share/ca-certificates/:/etc/ssl/certs -p 4001:4001 -p 2380:2380 -p 2379:2379 \
		--name args-etcd quay.io/coreos/etcd \
		-name etcd0 \
		-advertise-client-urls http://${ARGS_DOCKER_HOST}:2379,http://${ARGS_DOCKER_HOST}:4001 \
		-listen-client-urls http://0.0.0.0:2379,http://0.0.0.0:4001 \
		-initial-advertise-peer-urls http://${ARGS_DOCKER_HOST}:2380 \
		-listen-peer-urls http://0.0.0.0:2380 \
		-initial-cluster-token etcd-cluster-1 \
		-initial-cluster etcd0=http://${ARGS_DOCKER_HOST}:2380 \
		-initial-cluster-state new; \
	elif [ $(shell docker ps | grep -ci args-etcd) -eq 0 ]; then \
		echo restarting args-etcd; \
		docker start args-etcd > /dev/null; \
	fi

stop-docker:
	@if [ $(shell docker ps -a | grep -ci args-etcd) -eq 1 ]; then \
		echo Stopping Container args-etcd; \
		docker stop args-etcd > /dev/null; \
	fi

test:
	@echo Running Tests
	@go test .

all: start-docker test

