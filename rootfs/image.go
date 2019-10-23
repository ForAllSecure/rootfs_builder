package rootfs

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"

	"github.com/ForAllSecure/rootfs_builder/util"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/partial"
	"github.com/pkg/errors"
)

// Spec for rootfs extraction
type Spec struct {
	// Destination to extract to
	Dest string
	// User to chown files in rootfs to
	User string
	// Use the subuid associated with the given user for chowning
	UseSubuid bool
	subuid    int
	subgid    int
}

// PulledImage using provided PullableImage
type PulledImage struct {
	// User specified requirements for rootfs
	spec Spec
	name string
	img  v1.Image
}

// Digest from pulled image
func (pulledImg *PulledImage) Digest() (string, error) {
	// Digest() fails silently on images older than June 2016 (i.e. returns a
	// random hash that changes), so check img age
	_, err := getConfig(pulledImg.img)
	if err != nil {
		return "", err
	}
	hash, err := pulledImg.img.Digest()
	if err != nil {
		return "", errors.WithStack(err)
	}
	ref, err := name.ParseReference(pulledImg.name, name.WeakValidation)
	if err != nil {
		return "", errors.WithStack(err)
	}
	buf := fmt.Sprintf("%s/%s@%s\n", ref.Context().RegistryStr(), ref.Context().RepositoryStr(), hash.String())

	return buf, nil
}

// Extract rootfs
func (pulledImg *PulledImage) Extract() error {
	// Ensure we have a valid location to extract to
	err := pulledImg.validateDest()
	if err != nil {
		return err
	}

	// Dump the config
	err = pulledImg.writeConfig()
	if err != nil {
		return err
	}

	// Get a list of layers
	layers, err := pulledImg.img.Layers()
	if err != nil {
		return err
	}

	rootfsPath := filepath.Join(pulledImg.spec.Dest, "rootfs")
	if err := os.MkdirAll(rootfsPath, 0755); err != nil {
		return err
	}

	if err := pulledImg.validateUser(); err != nil {
		return err
	}

	// Extract the layers
	for _, layer := range layers {
		err = extractLayer(layer, rootfsPath, pulledImg.spec.subuid, pulledImg.spec.subgid)
		if err != nil {
			return err
		}
	}

	if err := os.Chown(rootfsPath, pulledImg.spec.subuid, pulledImg.spec.subuid); err != nil {
		return err
	}

	return nil
}

// Confirm that the user exists, and look up the appropriate subuid/subgid
func (pulledImg *PulledImage) validateUser() error {
	// Default to current user
	userObj, err := user.Current()

	// The config provided a user
	if pulledImg.spec.User != "" {
		userObj, err = user.Lookup(pulledImg.spec.User)
	}

	// Failed to find the user
	if err != nil {
		return err
	}

	// Get subuids for user namespace
	subuid, err := strconv.Atoi(userObj.Uid)
	if err != nil {
		return err
	}
	subgid, err := strconv.Atoi(userObj.Gid)
	if err != nil {
		return err
	}
	if pulledImg.spec.UseSubuid {
		subuid, subgid, err = util.GetSubid(userObj)
		if err != nil {
			return err
		}
	}

	pulledImg.spec.subuid = subuid
	pulledImg.spec.subgid = subgid

	return nil
}

// Validate the output location for the rootfs
func (pulledImg *PulledImage) validateDest() error {
	if pulledImg.spec.Dest == "" {
		return errors.New("Specify output destination for rootfs")
	}
	// Create the directory if it doesn't exist
	if _, err := os.Stat(pulledImg.spec.Dest); os.IsNotExist(err) {
		_ = os.Mkdir(pulledImg.spec.Dest, 0755)
	}
	return nil
}

// extract config.json from image and write to image.Dest.
// assumes image.Dest is valid.
func (pulledImg *PulledImage) writeConfig() error {
	configFile, err := getConfig(pulledImg.img)
	if err != nil {
		return err
	}
	jdata, err := json.MarshalIndent(configFile, "", " ")
	if err != nil {
		return err
	}
	configPath := filepath.Join(pulledImg.spec.Dest, "config.json")
	jsonFile, err := os.Create(configPath)
	if err != nil {
		return err
	}
	_, err = jsonFile.Write(jdata)
	return err
}

// extract config.json from image and check for errors
func getConfig(img partial.WithConfigFile) (*v1.ConfigFile, error) {
	configFile, err := img.ConfigFile()
	if err != nil {
		return nil, errors.Wrap(err, "could not retrieve config from image")
	}
	return configFile, nil
}
