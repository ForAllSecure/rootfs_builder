.PHONY: static dev build_in_container

local_static:
	go build -ldflags "-linkmode external -extldflags -static" -tags="netgo osusergo" -o rootfs_builder -a main.go

static:
	docker run --privileged -it -v `pwd`:/rootfs_builder golang:1.12 bash -c "cd /rootfs_builder && make local_static"

dev: rootfs_image
	docker run -it --privileged -v `pwd`:/rootfs_builder rootfs_image bash -c "cd /rootfs_builder; bash"

local_build:
	go build -o rootfs_builder main.go

rootfs_image:
	docker build -t rootfs_image .

local_test: local_build
	./test/integration.sh

test: rootfs_image
	docker run -it --privileged -v `pwd`:/rootfs_builder rootfs_image
