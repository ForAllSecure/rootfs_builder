FROM golang:1.12

WORKDIR /rootfs_builder
ADD . .
RUN make local_build
ENTRYPOINT make local_test
