FROM golang:1.12

WORKDIR /rootfs_builder
ADD . .
RUN make local_buildL
CMD make local_test
