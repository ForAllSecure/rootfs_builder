#!/bin/bash
#
# Pull apline:3.10, extract the rootfs, and verify its hash

set -xe

# set up
test_dir="/test/"
rm -rf $test_dir

# run
./rootfs_builder test/config.json

# check config hash
config_md5=`md5sum $test_dir/config.json | head -n1 | awk '{print $1;}'`
correct_config_md5="559b3bc2fd267ad4f75900f07763511a"
if [ "$config_md5" != "$correct_config_md5" ]; then
    echo "configs don't match"
    exit 1
fi

# check rootfs hash
rootfs_md5=`find $test_dir/rootfs -type f -exec md5sum {} \; | sort -k 2 | md5sum | head -n1 | awk '{print $1;}'`
correct_rootfs_md5="483a1dff1f39a232b3a876a99f9f8cd4"
echo $rootfs_md5
if [ "$rootfs_md5" != "$correct_rootfs_md5" ]; then
    echo "rootfs doesn't match"
    exit 1
fi

# tear down
rm -rf $test_dir
