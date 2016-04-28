package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/docker/distribution"
	"github.com/docker/distribution/context"
	"github.com/docker/distribution/digest"
	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/distribution/manifest/schema2"
	regapi "github.com/docker/distribution/registry/api/v2"
	repomw "github.com/docker/distribution/registry/middleware/repository"
	"github.com/docker/libtrust"

	kapi "k8s.io/kubernetes/pkg/api"
	kerrors "k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/client/restclient"
	kclient "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/util/sets"

	"github.com/openshift/origin/pkg/client"
	imageapi "github.com/openshift/origin/pkg/image/api"
	"github.com/openshift/origin/pkg/image/importer"
)

var (
	// cachedLayers is a shared cache of blob digests to remote repositories that have previously
	// been identified as containing that blob. Thread safe and reused by all middleware layers.
	cachedLayers digestToRepositoryCache
	// secureTransport is the transport pool used for pullthrough to remote registries marked as
	// secure.
	secureTransport http.RoundTripper
	// insecureTransport is the transport pool that does not verify remote TLS certificates for use
	// during pullthrough against registries marked as insecure.
	insecureTransport http.RoundTripper
)

func init() {
	cache, err := newDigestToRepositoryCache(1024)
	if err != nil {
		panic(err)
	}
	cachedLayers = cache

	// load the client when the middleware is initialized, which allows test code to change
	// DefaultRegistryClient before starting a registry.
	repomw.Register("openshift",
		func(ctx context.Context, repo distribution.Repository, options map[string]interface{}) (distribution.Repository, error) {
			registryClient, quotaClient, err := DefaultRegistryClient.Clients()
			if err != nil {
				return nil, err
			}
			return newRepositoryWithClient(registryClient, quotaClient, ctx, repo, options)
		},
	)

	secureTransport = http.DefaultTransport
	insecureTransport, err = restclient.TransportFor(&restclient.Config{Insecure: true})
	if err != nil {
		panic(fmt.Sprintf("Unable to configure a default transport for importing insecure images: %v", err))
	}
}

// repository wraps a distribution.Repository and allows manifests to be served from the OpenShift image
// API.
type repository struct {
	distribution.Repository

	ctx            context.Context
	quotaClient    kclient.ResourceQuotasNamespacer
	registryClient client.Interface
	registryAddr   string
	namespace      string
	name           string

	// if true, the repository will check remote references in the image stream to support pulling "through"
	// from a remote repository
	pullthrough bool
	// allowSchema2 allows to refuse the manifest schema version 2
	allowSchema2 bool
	// cachedLayers remembers a mapping of layer digest to repositories recently seen with that image to avoid
	// having to check every potential upstream repository when a blob request is made. The cache is useful only
	// when session affinity is on for the registry, but in practice the first pull will fill the cache.
	cachedLayers digestToRepositoryCache
}

var _ distribution.ManifestService = &repository{}

// newRepositoryWithClient returns a new repository middleware.
func newRepositoryWithClient(registryClient client.Interface, quotaClient kclient.ResourceQuotasNamespacer, ctx context.Context, repo distribution.Repository, options map[string]interface{}) (distribution.Repository, error) {
	registryAddr := os.Getenv("DOCKER_REGISTRY_URL")
	if len(registryAddr) == 0 {
		return nil, errors.New("DOCKER_REGISTRY_URL is required")
	}

	pullthrough := getBoolOption("pullthrough", false, options)
	allowSchema2 := getBoolOption("allowSchema2", false, options)

	nameParts := strings.SplitN(repo.Named().Name(), "/", 2)
	if len(nameParts) != 2 {
		return nil, fmt.Errorf("invalid repository name %q: it must be of the format <project>/<name>", repo.Named().Name())
	}

	return &repository{
		Repository: repo,

		ctx:            ctx,
		quotaClient:    quotaClient,
		registryClient: registryClient,
		registryAddr:   registryAddr,
		namespace:      nameParts[0],
		name:           nameParts[1],
		pullthrough:    pullthrough,
		allowSchema2:   allowSchema2,
		cachedLayers:   cachedLayers,
	}, nil
}

func getBoolOption(name string, defval bool, options map[string]interface{}) bool {
	if value, ok := options[name]; ok {
		if b, ok := value.(bool); ok {
			return b
		}
	}
	return defval
}

// Manifests returns r, which implements distribution.ManifestService.
func (r *repository) Manifests(ctx context.Context, options ...distribution.ManifestServiceOption) (distribution.ManifestService, error) {
	context.GetLogger(r.ctx).Debugf("(repository).Manifests: starting")
	if r.ctx == ctx {
		return r, nil
	}
	repo := repository(*r)
	repo.ctx = ctx
	return &repo, nil
}

// Blobs returns a blob store which can delegate to remote repositories.
func (r *repository) Blobs(ctx context.Context) distribution.BlobStore {
	context.GetLogger(r.ctx).Debugf("(repository).Blobs: starting")
	repo := repository(*r)
	repo.ctx = ctx

	bs := &quotaRestrictedBlobStore{
		BlobStore: r.Repository.Blobs(ctx),
		repo:      &repo,
	}
	if !r.pullthrough {
		return bs
	}

	return &pullthroughBlobStore{
		BlobStore: bs,

		repo:          &repo,
		digestToStore: make(map[string]distribution.BlobStore),
	}
}

// Tags returns a reference to this repositories tag service.
func (r *repository) Tags(ctx context.Context) distribution.TagService {
	context.GetLogger(r.ctx).Debugf("(repository).Tags: starting")
	imageStream, err := r.getImageStream()
	if err != nil {
		context.GetLogger(r.ctx).Errorf("Error retrieving ImageStream %s/%s: %v", r.namespace, r.name, err)
		return nil
	}
	return &tagService{
		TagService:     r.Repository.Tags(ctx),
		registryClient: r.registryClient,
		imageStream:    imageStream,
	}
}

// Exists returns true if the manifest specified by dgst exists.
func (r *repository) Exists(ctx context.Context, dgst digest.Digest) (bool, error) {
	context.GetLogger(r.ctx).Debugf("(repository).Exists: starting")
	image, err := r.getImage(dgst)
	if err != nil {
		return false, err
	}
	return image != nil, nil
}

// Get retrieves the manifest with digest `dgst`.
func (r *repository) Get(ctx context.Context, dgst digest.Digest, options ...distribution.ManifestServiceOption) (distribution.Manifest, error) {
	context.GetLogger(r.ctx).Debugf("(repository).Get: starting %s", dgst.String())

	if _, err := r.getImageStreamImage(dgst); err != nil {
		context.GetLogger(r.ctx).Errorf("Error retrieving ImageStreamImage %s/%s@%s: %v", r.namespace, r.name, dgst.String(), err)
		return nil, err
	}

	image, err := r.getImage(dgst)
	if err != nil {
		context.GetLogger(r.ctx).Errorf("Error retrieving image %s: %v", dgst.String(), err)
		return nil, err
	}

	supportsSchema2 := r.isSchema2Supported(ctx)

	ref := imageapi.DockerImageReference{Namespace: r.namespace, Name: r.name, Registry: r.registryAddr}
	manifest, err := r.manifestFromImageWithCachedLayers(image, ref.DockerClientDefaults().Exact(), supportsSchema2)

	return manifest, err
}

func (r *repository) isSchema2Supported(ctx context.Context) bool {
	if req, err := context.GetRequest(ctx); err == nil {
		if acceptHeaders, ok := req.Header["Accept"]; ok {
			for _, mediaType := range acceptHeaders {
				if mediaType == schema2.MediaTypeManifest {
					return true
				}
			}
		}
	} else {
		context.GetLogger(r.ctx).Errorf("Unable to get http request: %v", err)
	}
	return false
}

// Put creates or updates the named manifest.
func (r *repository) Put(ctx context.Context, manifest distribution.Manifest, options ...distribution.ManifestServiceOption) (digest.Digest, error) {
	context.GetLogger(r.ctx).Debugf("(repository).Put: starting")
	switch manifest.(type) {
	case *schema1.SignedManifest:
	case *schema2.DeserializedManifest:
	default:
		err := fmt.Errorf("unrecognized manifest type %T", manifest)
		return "", regapi.ErrorCodeManifestInvalid.WithDetail(err)
	}

	if !r.allowSchema2 {
		if _, ok := manifest.(*schema1.SignedManifest); !ok {
			err := fmt.Errorf("schema version 2 disabled")
			return "", regapi.ErrorCodeManifestInvalid.WithDetail(err)
		}
	}

	// Resolve the payload in the manifest.
	mediatype, payload, err := manifest.Payload()
	if err != nil {
		return digest.Digest(""), err
	}

	// Calculate digest
	dgst := digest.FromBytes(payload)

	// Upload to openshift
	ism := imageapi.ImageStreamMapping{
		ObjectMeta: kapi.ObjectMeta{
			Namespace: r.namespace,
			Name:      r.name,
		},
		Image: imageapi.Image{
			ObjectMeta: kapi.ObjectMeta{
				Name: dgst.String(),
				Annotations: map[string]string{
					imageapi.ManagedByOpenShiftAnnotation: "true",
				},
			},
			DockerImageReference:         fmt.Sprintf("%s/%s/%s@%s", r.registryAddr, r.namespace, r.name, dgst.String()),
			DockerImageManifest:          string(payload),
			DockerImageManifestMediaType: mediatype,
		},
	}

	for _, option := range options {
		if opt, ok := option.(distribution.WithTagOption); ok {
			ism.Tag = opt.Tag
			break
		}
	}

	if err = r.fillImageWithMetadata(manifest, &ism.Image); err != nil {
		return digest.Digest(""), err
	}

	if err := r.registryClient.ImageStreamMappings(r.namespace).Create(&ism); err != nil {
		// if the error was that the image stream wasn't found, try to auto provision it
		statusErr, ok := err.(*kerrors.StatusError)
		if !ok {
			context.GetLogger(r.ctx).Errorf("Error creating ImageStreamMapping: %s", err)
			return digest.Digest(""), err
		}

		if kerrors.IsForbidden(statusErr) {
			context.GetLogger(r.ctx).Errorf("Denied creating ImageStreamMapping: %v", statusErr)
			return digest.Digest(""), distribution.ErrAccessDenied
		}

		status := statusErr.ErrStatus
		if status.Code != http.StatusNotFound || (strings.ToLower(status.Details.Kind) != "imagestream" /*pre-1.2*/ && strings.ToLower(status.Details.Kind) != "imagestreams") || status.Details.Name != r.name {
			context.GetLogger(r.ctx).Errorf("Error creating ImageStreamMapping: %s", err)
			return digest.Digest(""), err
		}

		stream := imageapi.ImageStream{
			ObjectMeta: kapi.ObjectMeta{
				Name: r.name,
			},
		}

		client, ok := UserClientFrom(r.ctx)
		if !ok {
			context.GetLogger(r.ctx).Errorf("Error creating user client to auto provision image stream: Origin user client unavailable")
			return digest.Digest(""), statusErr
		}

		if _, err := client.ImageStreams(r.namespace).Create(&stream); err != nil {
			context.GetLogger(r.ctx).Errorf("Error auto provisioning image stream: %s", err)
			return digest.Digest(""), statusErr
		}

		// try to create the ISM again
		if err := r.registryClient.ImageStreamMappings(r.namespace).Create(&ism); err != nil {
			context.GetLogger(r.ctx).Errorf("Error creating image stream mapping: %s", err)
			return digest.Digest(""), err
		}
	}

	return dgst, nil
}

// fillImageWithMetadata fills a given image with metadata. Also correct layer sizes with blob sizes. Newer
// Docker client versions don't set layer sizes in the manifest at all. Origin master needs correct layer
// sizes for proper image quota support. That's why we need to fill the metadata in the registry.
func (r *repository) fillImageWithMetadata(manifest distribution.Manifest, image *imageapi.Image) error {
	if deserializedManifest, ok := manifest.(*schema2.DeserializedManifest); ok {
		configBytes, err := r.Blobs(r.ctx).Get(r.ctx, deserializedManifest.Config.Digest)
		if err != nil {
			context.GetLogger(r.ctx).Errorf("Failed to get image config %s: %v", deserializedManifest.Config.Digest.String(), err)
			return err
		}
		image.DockerConfigImage = string(configBytes)
	}

	if signedManifest, ok := manifest.(*schema1.SignedManifest); ok {
		signatures, err := signedManifest.Signatures()
		if err != nil {
			return err
		}

		for _, signDigest := range signatures {
			image.DockerImageSignatures = append(image.DockerImageSignatures, imageapi.DockerImageSignature(signDigest))
		}
	}

	if err := imageapi.ImageWithMetadata(image); err != nil {
		return err
	}

	refs := manifest.References()

	layerSet := sets.NewString()
	size := int64(0)

	blobs := r.Blobs(r.ctx)
	for i := range image.DockerImageLayers {
		layer := &image.DockerImageLayers[i]
		// DockerImageLayers represents manifest.Manifest.FSLayers in reversed order
		desc, err := blobs.Stat(r.ctx, refs[len(image.DockerImageLayers)-i-1].Digest)
		if err != nil {
			context.GetLogger(r.ctx).Errorf("Failed to stat blobs %s of image %s", layer.Name, image.DockerImageReference)
			return err
		}
		layer.Size = desc.Size
		// count empty layer just once (empty layer may actually have non-zero size)
		if !layerSet.Has(layer.Name) {
			size += desc.Size
			layerSet.Insert(layer.Name)
		}
	}

	image.DockerImageMetadata.Size = size
	context.GetLogger(r.ctx).Infof("Total size of image %s with docker ref %s: %d", image.Name, image.DockerImageReference, size)

	return nil
}

// Delete deletes the manifest with digest `dgst`. Note: Image resources
// in OpenShift are deleted via 'oadm prune images'. This function deletes
// the content related to the manifest in the registry's storage (signatures).
func (r *repository) Delete(ctx context.Context, dgst digest.Digest) error {
	ms, err := r.Repository.Manifests(r.ctx)
	if err != nil {
		return err
	}
	return ms.Delete(ctx, dgst)
}

// importContext loads secrets for this image stream and returns a context for getting distribution
// clients to remote repositories.
func (r *repository) importContext() importer.RepositoryRetriever {
	secrets, err := r.registryClient.ImageStreamSecrets(r.namespace).Secrets(r.name, kapi.ListOptions{})
	if err != nil {
		context.GetLogger(r.ctx).Errorf("Error getting secrets for repository %q: %v", r.Named().Name(), err)
		secrets = &kapi.SecretList{}
	}
	credentials := importer.NewCredentialsForSecrets(secrets.Items)
	return importer.NewContext(secureTransport, insecureTransport).WithCredentials(credentials)
}

// getImageStream retrieves the ImageStream for r.
func (r *repository) getImageStream() (*imageapi.ImageStream, error) {
	return r.registryClient.ImageStreams(r.namespace).Get(r.name)
}

// getImage retrieves the Image with digest `dgst`.
func (r *repository) getImage(dgst digest.Digest) (*imageapi.Image, error) {
	return r.registryClient.Images().Get(dgst.String())
}

// getImageStreamImage retrieves the Image with digest `dgst` for the ImageStream
// associated with r. This ensures the image belongs to the image stream.
func (r *repository) getImageStreamImage(dgst digest.Digest) (*imageapi.ImageStreamImage, error) {
	return r.registryClient.ImageStreamImages(r.namespace).Get(r.name, dgst.String())
}

// rememberLayers caches the provided layers
func (r *repository) rememberLayers(manifest distribution.Manifest, cacheName string) {
	if !r.pullthrough {
		return
	}
	// remember the layers in the cache as an optimization to avoid searching all remote repositories
	for _, layer := range manifest.References() {
		r.cachedLayers.RememberDigest(layer.Digest, cacheName)
	}
}

// manifestFromImageWithCachedLayers loads the image and then caches any located layers
func (r *repository) manifestFromImageWithCachedLayers(image *imageapi.Image, cacheName string, supportsSchema2 bool) (manifest distribution.Manifest, err error) {
	isManifest2 := image.DockerImageManifestMediaType == schema2.MediaTypeManifest

	if supportsSchema2 {
		if isManifest2 {
			manifest, err = r.deserializedManifestFromImage(image)
		} else {
			manifest, err = r.signedManifestFromImage(image)
		}
	} else {
		if isManifest2 {
			err = fmt.Errorf("Unable to convert new image to old one")
		} else {
			manifest, err = r.signedManifestFromImage(image)
		}
	}

	if err != nil {
		return
	}

	r.rememberLayers(manifest, cacheName)
	return
}

// manifestFromImage converts an Image to a SignedManifest.
func (r *repository) signedManifestFromImage(image *imageapi.Image) (*schema1.SignedManifest, error) {
	if image.DockerImageManifestMediaType == schema2.MediaTypeManifest {
		context.GetLogger(r.ctx).Errorf("old client pulling new image %s", image.DockerImageReference)
		return nil, fmt.Errorf("Unable to convert new image to old one")
	}

	dgst, err := digest.ParseDigest(image.Name)
	if err != nil {
		return nil, err
	}

	raw := []byte(image.DockerImageManifest)
	// prefer signatures from the manifest
	if _, err := libtrust.ParsePrettySignature(raw, "signatures"); err == nil {
		sm := schema1.SignedManifest{Canonical: raw}
		if err := json.Unmarshal(raw, &sm); err == nil {
			return &sm, nil
		}
	}

	var signBytes [][]byte
	if len(image.DockerImageSignatures) == 0 {
		// Fetch the signatures for the manifest
		signatures, err := r.getSignatures(dgst)
		if err != nil {
			return nil, err
		}

		for _, signatureDigest := range signatures {
			signBytes = append(signBytes, []byte(signatureDigest))
		}
	} else {
		for _, sign := range image.DockerImageSignatures {
			signBytes = append(signBytes, sign)
		}
	}

	jsig, err := libtrust.NewJSONSignature(raw, signBytes...)
	if err != nil {
		return nil, err
	}

	// Extract the pretty JWS
	raw, err = jsig.PrettySignature("signatures")
	if err != nil {
		return nil, err
	}

	var sm schema1.SignedManifest
	if err := json.Unmarshal(raw, &sm); err != nil {
		return nil, err
	}
	return &sm, err
}

func (r *repository) getSignatures(dgst digest.Digest) ([]digest.Digest, error) {
	// We can not use the r.repository here. docker/distribution wraps all the methods that
	// write or read blobs. It is made for notifications service. We need to get a real
	// repository without any wrappers.
	repository, err := Registry.Repository(r.ctx, r.Named())
	if err != nil {
		return nil, err
	}

	manifestService, err := repository.Manifests(r.ctx)
	if err != nil {
		return nil, err
	}

	signaturesGetter, ok := manifestService.(distribution.SignaturesGetter)
	if !ok {
		return nil, fmt.Errorf("Unable to convert ManifestService into SignaturesGetter")
	}

	return signaturesGetter.GetSignatures(r.ctx, dgst)
}

// deserializedManifestFromImage converts an Image to a DeserializedManifest.
func (r *repository) deserializedManifestFromImage(image *imageapi.Image) (*schema2.DeserializedManifest, error) {
	var manifest schema2.DeserializedManifest
	if err := json.Unmarshal([]byte(image.DockerImageManifest), &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}
