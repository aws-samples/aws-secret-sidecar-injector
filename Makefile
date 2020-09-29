DOCKER_REPOSITORY ?= docker.io/amazon
VERSION	?= $(shell cat VERSION)
INIT_CONTAINER_IMAGE := ${DOCKER_REPOSITORY}/aws-secrets-manager-secret-sidecar:${VERSION}
ADM_CONTROLLER_IMAGE := ${DOCKER_REPOSITORY}/aws-secrets-manager-secret-adm-controller:${VERSION}

.PHONY: build initcontainer admissioncontroller publish

build: initcontainer admissioncontroller

initcontainer:
	docker build --build-arg APP=initcontainer -t ${INIT_CONTAINER_IMAGE} .

admissioncontroller:
	docker build --build-arg APP=admissioncontroller -t ${ADM_CONTROLLER_IMAGE} .

publish: build
	docker push $(INIT_CONTAINER_IMAGE)
	docker push $(ADM_CONTROLLER_IMAGE)


