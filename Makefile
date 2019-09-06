.PHONY: static dev build_in_container

static:
	go build -ldflags "-linkmode external -extldflags -static" -tags="netgo osusergo" -o rootfs_builder -a main.go

# Statically compile inside a container
in_container:
	docker run --privileged -it -v `pwd`:/rootfs_builder golang:1.12 bash -c "cd /rootfs_builder && make static"

dev:
	docker run -it --privileged -v `pwd`:/rootfs_builder golang:1.12 bash -c "cd /rootfs_builder && bash"
