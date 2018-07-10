TAG?=latest-dev
.PHONY: build

build:
	docker build -t alexellis/faas-idler:${TAG} .
push:
	docker push alexellis/faas-idler:${TAG}
