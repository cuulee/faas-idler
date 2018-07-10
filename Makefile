TAG?=latest-dev
.PHONY: build

build:
	docker build -t alexellis/faas-idler:${TAG} .
