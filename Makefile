.PHONY: whiteout_image rootfs pull

whiteout_image:
	docker build -f dockerfiles/whiteout.dockerfile -t whiteout .
	docker save -o image.tar whiteout
	-rm -r simple-image
	mkdir simple-image
	tar -xvf image.tar -C simple-image

# Pull image and extract rootfs
rootfs:
	docker run -it --entrypoint=bash --rm -v `pwd`:/go/src/local -w /go/src/local golang:1.11.1 \
		-c "go get github.com/google/go-containerregistry/pkg/authn; \
		go get github.com/pkg/errors; \
		go get github.com/google/go-containerregistry/pkg/name; \
		go get github.com/google/go-containerregistry/pkg/v1; \
		go get github.com/google/go-containerregistry/pkg/v1/remote; \
		go run rootfs_builder.go alpine:latest alpine_rootfs; bash"

debug_container:
	docker run -it --privileged -v `pwd`:/rootfs golang:1.12 bash
