// Parts of this file are modified from Kaniko, an Apache 2.0 licensed project,
// and so this copyright applies.
//
// Copyright 2018 Google LLC
//
// https://github.com/GoogleContainerTools/kaniko/blob/master/pkg/util/fs_util.go
// Commit # 3422d55

package rootfs

import (
	"archive/tar"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/ForAllSecure/rootfs_builder/log"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/v1util"
	"github.com/pkg/errors"
)

// extract a single file
func extractFile(dest string, hdr *tar.Header, tr io.Reader, subuid int, subgid int) error {
	// Construct filepath from tar header
	path := filepath.Join(dest, filepath.Clean(hdr.Name))
	dir := filepath.Dir(path)

	// Get metadata from tar header
	mode := hdr.FileInfo().Mode()
	uid := hdr.Uid + subuid
	gid := hdr.Gid + subgid

	switch hdr.Typeflag {
	case tar.TypeReg:
		// It's possible a file is in the tar before it's directory.
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}
		}
		// Check if something already exists at path (symlinks etc.)
		// If so, delete it
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			if err := os.RemoveAll(path); err != nil {
				return errors.Wrapf(err, "error removing %s to make way for new file.", path)
			}
		}
		currFile, err := os.Create(path)
		if err != nil {
			return err
		}
		// manually set permissions on file, since the default umask (022) will interfere
		if err = os.Chmod(path, mode); err != nil {
			return err
		}
		if _, err = io.Copy(currFile, tr); err != nil {
			return err
		}
		if err = currFile.Chown(uid, gid); err != nil {
			return err
		}
		currFile.Close()
	case tar.TypeDir:
		if err := os.MkdirAll(path, mode); err != nil {
			return err
		}
		// In some cases, MkdirAll doesn't change the permissions, so run Chmod
		if err := os.Chmod(path, mode); err != nil {
			return err
		}
		if err := os.Chown(path, uid, gid); err != nil {
			return err
		}

	// Hard link: Two files point to same data on disc.  Assume OFS/Docker orders tarball such
	// that hard link comes after regular file that hard link points to.
	case tar.TypeLink:
		// The base directory for a link may not exist before it is created.
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		// Check if something already exists at path
		// If so, delete it
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			if err := os.RemoveAll(path); err != nil {
				return errors.Wrapf(err, "error removing %s to make way for new link", hdr.Name)
			}
		}
		// Link hard link to its target
		link := filepath.Clean(filepath.Join(dest, hdr.Linkname))
		if err := os.Link(link, path); err != nil {
			return err
		}

	case tar.TypeSymlink:
		// The base directory for a symlink may not exist before it is created.
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		// Check if something already exists at path
		// If so, delete it
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			if err := os.RemoveAll(path); err != nil {
				return errors.Wrapf(err, "error removing %s to make way for new symlink", hdr.Name)
			}
		}
		if err := os.Symlink(hdr.Linkname, path); err != nil {
			return err
		}
		if err := os.Lchown(path, uid, gid); err != nil {
			return err
		}
	}
	return nil
}

// Whiteouts
func whiteout(tr *tar.Reader, rootfs string) error {
	// Iterate through headers, removing whiteouts first
	for {
		hdr, err := tr.Next()
		// Done with this tar layer
		if err == io.EOF {
			break
		}
		// Something went wrong
		if err != nil {
			return err
		}
		path := filepath.Join(rootfs, filepath.Clean(hdr.Name))
		base := filepath.Base(path)
		dir := filepath.Dir(path)
		// Opaque directory
		if strings.HasPrefix(base, ".wh..wh..opq") {
			if err := os.RemoveAll(dir); err != nil {
				return errors.Wrapf(err, "removing whiteout %s", hdr.Name)
			}
		} else if strings.HasPrefix(base, ".wh.") {
			name := strings.TrimPrefix(base, ".wh.")
			if err := os.RemoveAll(filepath.Join(dir, name)); err != nil {
				return errors.Wrapf(err, "removing whiteout %s", hdr.Name)
			}
		} else {
			continue
		}
	}
	return nil
}

// Handle regular files
func handleFiles(tr *tar.Reader, rootfs string, subuid int, subgid int) error {
	// Iterate through the headers, extracting regular files
	for {
		hdr, err := tr.Next()
		// Done with this tar layer
		if err == io.EOF {
			break
		}
		// Something went wrong
		if err != nil {
			return err
		}
		path := filepath.Join(rootfs, filepath.Clean(hdr.Name))
		base := filepath.Base(path)
		// This is a whiteout file/directory, skip!
		if strings.HasPrefix(base, ".wh.") {
			continue
		}
		if err := extractFile(rootfs, hdr, tr, subuid, subgid); err != nil {
			return err
		}
	}
	return nil
}

// Get a tar reader from a v1.Layer
func tarReader(layer_file *os.File) (*tar.Reader, error) {
	r, err := v1util.GunzipReadCloser(layer_file)
	if err != nil {
		return nil, err
	}
	tr := tar.NewReader(r)
	return tr, nil
}

// saveLayer saves a Layer to disk
func saveLayer(layer v1.Layer) (*os.File, error) {
	digest, err := layer.Digest()
	if err != nil {
		return nil, err
	}

	layer_file, err := ioutil.TempFile("", fmt.Sprintf("%s", digest))
	if err != nil {
		return nil, errors.Wrapf(err, "generating tempfile")
	}

	rc, err := layer.Compressed()
	if err != nil {
		defer os.Remove(layer_file.Name())
		return nil, err
	}

	_, err = io.Copy(layer_file, rc)
	if err != nil {
		defer os.Remove(layer_file.Name())
		return nil, err
	}
	layer_file.Seek(0, 0)
	return layer_file, nil
}

// extractLayer accepts an open file descriptor to tarball and the destianation
// to extract the rootfs to
func extractLayer(layer v1.Layer, rootfs string, subuid int, subgid int) error {
	digest, err := layer.Digest()
	if err != nil {
		return err
	}
	size, err := layer.Size()
	if err != nil {
		return err
	}

	log.Debugf("Downloading layer %s, %d bytes", digest, size)
	layer_file, err := saveLayer(layer)
	if err != nil {
		return err
	}
	defer os.Remove(layer_file.Name())

	tr, err := tarReader(layer_file)
	if err != nil {
		return err
	}
	log.Debugf("Whiting out layer %s", digest)
	err = whiteout(tr, rootfs)
	if err != nil {
		return err
	}

	layer_file.Seek(0, 0)
	tr, err = tarReader(layer_file)
	if err != nil {
		return err
	}

	log.Debugf("Extracting layer %s", digest)
	err = handleFiles(tr, rootfs, subuid, subgid)
	if err != nil {
		return err
	}
	return nil
}
