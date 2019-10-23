package main

import (
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
		log.Errorf("Failed to initialize image from config: %+v", err)
		os.Exit(1)
	}
	pulledImage, err := pullableImage.Pull()
	if err != nil {
		log.Errorf("Failed to pull image: %+v", err)
		os.Exit(1)
	}
	// Extract rootfs
	if len(os.Args) == 2 {
		err = pulledImage.Extract()
		if err != nil {
			log.Errorf("Failed to extract rootfs: %+v", err)
			os.Exit(1)
		}
	} else {
		// Digest only
		digest, err := pulledImage.Digest()
		if err != nil {
			log.Errorf("Failed to get digest: %+v", err)
			os.Exit(1)
		}
		log.Info(digest)
	}
}
