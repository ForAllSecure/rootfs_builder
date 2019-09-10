package main

import (
	"fmt"
	"os"

	"github.com/ForAllSecure/rootfs_builder/log"
	"github.com/ForAllSecure/rootfs_builder/rootfs"
)

func main() {
	if len(os.Args) > 3 || len(os.Args) < 2 {
		log.Fatal("Usage: rootfs_builder <config.json>\n" +
			"\t\t\t\t\t--digest-only: only print the digest")
	}
	// Initialize pullable image from config
	pullableImage, err := rootfs.NewPullableImage(os.Args[1])
	if err != nil {
		log.Fatalf("%v", err.Error())
	}
	pulledImage, err := pullableImage.Pull()
	if err != nil {
		log.Fatalf("%v", err.Error())
	}
	// Extract rootfs
	if len(os.Args) == 2 {
		err = pulledImage.Extract()
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
	} else {
		// Digest only
		digest, err := pulledImage.Digest()
		if err != nil {
			log.Fatalf("%v", err.Error())
		}
		log.Info(digest)
	}
}
