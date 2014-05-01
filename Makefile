.PHONY: all build binary default db run clean cleandb

GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
DOCKER_IMAGE := enforcer:$(GIT_BRANCH)
DOCKER_RUN_ENFORCER := docker run --rm -i -t --privileged -e TESTFLAGS -v $(CURDIR)/bundles:/go/src/github.com/appstash/enforcer/bundles

default: binary

all: build
	$(DOCKER_RUN_ENFORCER) "$(DOCKER_IMAGE)" hack/make.sh

#all: clean build run logs


binary: build
		$(DOCKER_RUN_ENFORCER) "$(DOCKER_IMAGE)"  hack/make.sh binary

content:
	 	$(DOCKER_RUN_ENFORCER) "$(DOCKER_IMAGE)"  hack/make.sh content
ubuntu: binary
		$(DOCKER_RUN_ENFORCER) "$(DOCKER_IMAGE)"  hack/make.sh ubuntu
db:
		docker run -d -t -p 28015:28015 -p 8081:8080 -name rethinkdb crosbymichael/rethinkdb --bind all
cross: build
		$(DOCKER_RUN_ENFORCER) "$(DOCKER_IMAGE)" hack/make.sh binary cross
gox: build
		$(DOCKER_RUN_ENFORCER) "$(DOCKER_IMAGE)" hack/make.sh gox
shell: build
		$(DOCKER_RUN_ENFORCER) "$(DOCKER_IMAGE)" bash
container: build cleandb db
	$(DOCKER_RUN_ENFORCER) -h enforcer --link rethinkdb:db --name enforcer -p 4321:4321 "$(DOCKER_IMAGE)" hack/run.sh
clean:
	docker rm $(docker ps -a -q) ;  docker rmi enforcer:master
cleandb:
	docker stop rethinkdb ; docker rm rethinkdb || true
logs:
	watch -d docker logs enforcer
build: bundles
	docker build --rm -t "$(DOCKER_IMAGE)" .
bundles:
	mkdir bundles
