package rootfs

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"math"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/ForAllSecure/rootfs_builder/log"
	"github.com/ForAllSecure/rootfs_builder/util"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
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
	// Metadata for rootfs extraction
	Spec  Spec
	https bool
}

// MaxBackoff is the maximum backoff time per retry in seconds
var MaxBackoff float64 = 30

// DefaultRetries is the default number of retries
var DefaultRetries int = 3

// NewPullableImage initializes a PullableImage spec from a user provided config
func NewPullableImage(path string) (*PullableImage, error) {
	var pullableImage PullableImage
	err := util.UnmarshalFile(path, &pullableImage)
	if err != nil {
		return nil, err
	}
	if pullableImage.Retries <= 0 {
		pullableImage.Retries = DefaultRetries
	}
	pullableImage.https = true
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
		if strings.Contains(err.Error(), "http: server gave HTTP response to HTTPS client") {
			log.Info("Retrying with HTTP")
			pullable.https = false
		}
		// This is a v1 schema, give up early
		if strings.Contains(err.Error(), "unsupported MediaType") {
			err = errors.WithMessage(err, "Image is v1 schema and too old to support")
			break
		}
		// Either we are unauthorized, or this is a bad registry/image name
		if strings.Contains(err.Error(), "UNAUTHORIZED: authentication required") {
			break
		}
		// If we get a i/o timeout, it's either intermittent network failure
		// or an incorrect ip address etc. This means we've already failed 5
		// retries internal to go-containerregistry, so fail
		if strings.Contains(err.Error(), "i/o timeout") {
			log.Warnf("Connection to server timed out %s", err)
			break
		}
		switch err := errors.Cause(err).(type) {
		case *transport.Error:
			break
		default:
			log.Warnf("Unrecognized error: %s Trying again", err)
		}

		backoff := math.Pow(2, float64(i))
		backoff = math.Min(backoff, MaxBackoff)
		time.Sleep(time.Second * time.Duration(backoff))
	}
	// Failed to pull, return an error
	if err != nil {
		return nil, err
	}
	// Initialize the image
	pulled := &PulledImage{
		img:  img,
		name: pullable.Name,
		spec: pullable.Spec,
	}
	return pulled, nil
}

// pull a v1.image
func (pullable *PullableImage) pull() (v1.Image, error) {
	log.Debugf("Getting manifest for %s", pullable.Name)
	ref, err := name.ParseReference(pullable.Name, name.WeakValidation)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	registryName := ref.Context().RegistryStr()

	var newReg name.Registry
	if pullable.https {
		newReg, err = name.NewRegistry(registryName, name.WeakValidation)
	} else {
		newReg, err = name.NewRegistry(registryName, name.Insecure)
	}

	if err != nil {
		return nil, errors.WithStack(err)
	}

	if tag, ok := ref.(name.Tag); ok {
		tag.Repository.Registry = newReg
		ref = tag
	}
	if digest, ok := ref.(name.Digest); ok {
		digest.Repository.Registry = newReg
		ref = digest
	}

	transport := http.DefaultTransport.(*http.Transport)
	transport.DialContext = (&net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 10 * time.Second,
		DualStack: true,
	}).DialContext
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

		transport.TLSClientConfig = config
	}
	transportOption := remote.WithTransport(transport)

	authnOption := remote.WithAuthFromKeychain(authn.NewMultiKeychain(authn.DefaultKeychain))
	img, err := remote.Image(ref, transportOption, authnOption)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return img, nil
}
