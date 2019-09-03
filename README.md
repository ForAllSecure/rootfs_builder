Turn an image into a rootfs.

Note: The image should be the output of `docker save`.

Code recycled from Google's Kaniko, specifically:

rootfs builder:
https://github.com/GoogleContainerTools/kaniko/blob/master/pkg/util/fs_util.go

image pulling:
https://github.com/GoogleContainerTools/kaniko/blob/master/pkg/util/image_util.go#L96

Usage: go run main.go IMG DST

1. Install GO on osx: `brew install golang`
2. Build the whiteout image we will use to test with: `make whiteout_image`
3. Run main.go inside a container: `make rootfs` 


