FROM golang:1.14.3

WORKDIR /rootfs_builder
ADD . .
# This automatically adds a subuid mapping
RUN useradd fas
RUN make local_build
CMD make local_test
