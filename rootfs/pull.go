package rootfs

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"math"
	"net/http"
	"time"

	"github.com/ForAllSecure/rootfs_builder/log"
	"github.com/ForAllSecure/rootfs_builder/util"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/pkg/errors"
)

// PullableImage contains metadata necessary for pulling images
type PullableImage struct {
	// Name of image to pull
	Name string
	// Path to registry cert
	Cert *string
	// Number of attempts to retry pulling
	Retries int
	// Pull via https. Defaults to false
	HTTPS bool
	// Metadata for rootfs extraction
	Spec Spec
}

// NewPullableImage initializes a PullableImage spec from a user provided config
func NewPullableImage(path string) (*PullableImage, error) {
	var pullableImage PullableImage
	err := util.UnmarshalFile(path, &pullableImage)
	if err != nil {
		return nil, err
	}
	if pullableImage.Retries == 0 {
		pullableImage.Retries = 3
	}
	return &pullableImage, nil
}

// Pull a v1.Image and initialize a PulledImage struct to include the v1.img
// and metadata for extracting to a rootfs
func (pullable *PullableImage) Pull() (*PulledImage, error) {
	var err error
	var img v1.Image
	for i := 0; i < pullable.Retries; i++ {
		img, err = pullable.pull()
		if err == nil {
			break
		}
		backoff := math.Pow(2, float64(i))
		time.Sleep(time.Second * time.Duration(backoff))
	}
	// Failed to pull, return an error
	if err != nil {
		return nil, err
	}
	// Initialize the image
	pulled := &PulledImage{
		img:  img,
		spec: pullable.Spec,
	}
	return pulled, nil
}

// pull a v1.image
func (pullable *PullableImage) pull() (v1.Image, error) {
	log.Infof("pulling %s", pullable.Name)
	ref, err := name.ParseReference(pullable.Name, name.WeakValidation)
	if err != nil {
		return nil, err
	}
	registryName := ref.Context().RegistryStr()

	var newReg name.Registry
	if pullable.HTTPS {
		newReg, err = name.NewRegistry(registryName, name.WeakValidation)
	} else {
		newReg, err = name.NewRegistry(registryName, name.Insecure)
	}

	if err != nil {
		return nil, err
	}

	if tag, ok := ref.(name.Tag); ok {
		tag.Repository.Registry = newReg
		ref = tag
	}
	if digest, ok := ref.(name.Digest); ok {
		digest.Repository.Registry = newReg
		ref = digest
	}

	transport := http.DefaultTransport
	// A cert was provided
	if pullable.Cert != nil {
		rootCAs, err := x509.SystemCertPool()
		if err != nil {
			return nil, err
		}
		if rootCAs == nil {
			rootCAs = x509.NewCertPool()
		}
		// Read in the cert file
		certs, err := ioutil.ReadFile(*pullable.Cert)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read file %s to add to RootCAs", *pullable.Cert)
		}
		// Append our cert to the system pool
		if ok := rootCAs.AppendCertsFromPEM(certs); !ok {
			return nil, errors.Wrap(err, "Failed to append registry certificate")
		}

		// Trust the augmented cert pool in our client
		config := &tls.Config{
			RootCAs: rootCAs,
		}
		transport = &http.Transport{TLSClientConfig: config}

	}
	transportOption := remote.WithTransport(transport)

	authnOption := remote.WithAuthFromKeychain(authn.NewMultiKeychain(authn.DefaultKeychain))
	img, err := remote.Image(ref, transportOption, authnOption)
	if err != nil {
		return nil, err
	}
	return img, nil
}
