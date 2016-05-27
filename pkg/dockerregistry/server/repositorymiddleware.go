package server

import (
	"encoding/json"
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
	quotautil "github.com/openshift/origin/pkg/quota/util"
)

const (
	// DockerRegistryURLEnvVar is a mandatory environment variable name specifying url of internal docker
	// registry. All references to pushed images will be prefixed with its value.
	DockerRegistryURLEnvVar = "DOCKER_REGISTRY_URL"

	// EnforceQuotaEnvVar is a boolean environment variable that allows to turn quota enforcement on or off.
	// By default, quota enforcement is off. It overrides openshift middleware configuration option.
	// Recognized values are "true" and "false".
	EnforceQuotaEnvVar = "REGISTRY_MIDDLEWARE_REPOSITORY_OPENSHIFT_ENFORCEQUOTA"

	// ProjectCacheTTLEnvVar is an environment variable specifying an eviction timeout for project quota
	// objects. It takes a valid time duration string (e.g. "2m"). If empty, you get the default timeout. If
	// zero (e.g. "0m"), caching is disabled.
	ProjectCacheTTLEnvVar = "REGISTRY_MIDDLEWARE_REPOSITORY_OPENSHIFT_PROJECTCACHETTL"
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
	// quotaEnforcing contains shared caches of quota objects keyed by project name. Will be initialized
	// only if the quota is enforced. See EnforceQuotaEnvVar.
	quotaEnforcing *quotaEnforcingConfig
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
			if dockerRegistry == nil {
				panic(fmt.Sprintf("Configuration error: Middleware for registry not activated"))
			}

			if dockerStorageDriver == nil {
				panic(fmt.Sprintf("Configuration error: Middleware for storage driver not activated"))
			}

			registryOSClient, kClient, err := DefaultRegistryClient.Clients()
			if err != nil {
				return nil, err
			}
			if quotaEnforcing == nil {
				quotaEnforcing = newQuotaEnforcingConfig(ctx, os.Getenv(EnforceQuotaEnvVar), os.Getenv(ProjectCacheTTLEnvVar), options)
			}
			return newRepositoryWithClient(registryOSClient, kClient, kClient, ctx, repo, options)
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

	ctx              context.Context
	quotaClient      kclient.ResourceQuotasNamespacer
	limitClient      kclient.LimitRangesNamespacer
	registryOSClient client.Interface
	registryAddr     string
	namespace        string
	name             string

	// if true, the repository will check remote references in the image stream to support pulling "through"
	// from a remote repository
	pullthrough bool
	// acceptschema2 allows to refuse the manifest schema version 2
	acceptschema2 bool
	// cachedLayers remembers a mapping of layer digest to repositories recently seen with that image to avoid
	// having to check every potential upstream repository when a blob request is made. The cache is useful only
	// when session affinity is on for the registry, but in practice the first pull will fill the cache.
	cachedLayers digestToRepositoryCache
}

var _ distribution.ManifestService = &repository{}

// newRepositoryWithClient returns a new repository middleware.
func newRepositoryWithClient(
	registryOSClient client.Interface,
	quotaClient kclient.ResourceQuotasNamespacer,
	limitClient kclient.LimitRangesNamespacer,
	ctx context.Context,
	repo distribution.Repository,
	options map[string]interface{},
) (distribution.Repository, error) {
	registryAddr := os.Getenv(DockerRegistryURLEnvVar)
	if len(registryAddr) == 0 {
		return nil, fmt.Errorf("%s is required", DockerRegistryURLEnvVar)
	}

	pullthrough := getBoolOption("pullthrough", false, options)
	acceptschema2 := getBoolOption("acceptschema2", false, options)

	nameParts := strings.SplitN(repo.Named().Name(), "/", 2)
	if len(nameParts) != 2 {
		return nil, fmt.Errorf("invalid repository name %q: it must be of the format <project>/<name>", repo.Named().Name())
	}

	return &repository{
		Repository: repo,

		ctx:              ctx,
		quotaClient:      quotaClient,
		limitClient:      limitClient,
		registryOSClient: registryOSClient,
		registryAddr:     registryAddr,
		namespace:        nameParts[0],
		name:             nameParts[1],
		pullthrough:      pullthrough,
		acceptschema2:    acceptschema2,
		cachedLayers:     cachedLayers,
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
	if r.ctx == ctx {
		return r, nil
	}
	repo := repository(*r)
	repo.ctx = ctx
	return &repo, nil
}

// Blobs returns a blob store which can delegate to remote repositories.
func (r *repository) Blobs(ctx context.Context) distribution.BlobStore {
	repo := repository(*r)
	repo.ctx = ctx

	bs := r.Repository.Blobs(ctx)

	if quotaEnforcing != nil {
		bs = &quotaRestrictedBlobStore{
			BlobStore: bs,

			repo: &repo,
		}
	}

	if r.pullthrough {
		bs = &pullthroughBlobStore{
			BlobStore: bs,

			repo:          &repo,
			digestToStore: make(map[string]distribution.BlobStore),
		}
	}

	return bs
}

// Tags returns a reference to this repository tag service.
func (r *repository) Tags(ctx context.Context) distribution.TagService {
	return &tagService{
		TagService:     r.Repository.Tags(ctx),
		repository:     r,
		registryClient: r.registryOSClient,
		namespace:      r.namespace,
		name:           r.name,
		pullthrough:    r.pullthrough,
	}
}

// Exists returns true if the manifest specified by dgst exists.
func (r *repository) Exists(ctx context.Context, dgst digest.Digest) (bool, error) {
	image, err := r.getImage(dgst)
	if err != nil {
		return false, err
	}
	return image != nil, nil
}

// Get retrieves the manifest with digest `dgst`.
func (r *repository) Get(ctx context.Context, dgst digest.Digest, options ...distribution.ManifestServiceOption) (distribution.Manifest, error) {
	if _, err := r.getImageStreamImage(dgst); err != nil {
		context.GetLogger(r.ctx).Errorf("error retrieving ImageStreamImage %s/%s@%s: %v", r.namespace, r.name, dgst.String(), err)
		return nil, err
	}

	image, err := r.getImage(dgst)
	if err != nil {
		context.GetLogger(r.ctx).Errorf("error retrieving image %s: %v", dgst.String(), err)
		return nil, err
	}

	supportsSchema2 := r.isSchema2Supported(ctx)

	ref := imageapi.DockerImageReference{Namespace: r.namespace, Name: r.name, Registry: r.registryAddr}
	manifest, err := r.manifestFromImageWithCachedLayers(image, ref.DockerClientDefaults().Exact(), supportsSchema2)

	return manifest, err
}

func (r *repository) isSchema2Supported(ctx context.Context) bool {
	req, err := context.GetRequest(ctx)
	if err != nil {
		context.GetLogger(r.ctx).Errorf("unable to get http request: %v", err)
		return false
	}

	acceptHeaders, ok := req.Header["Accept"]
	if !ok {
		return false
	}

	for _, mediaType := range acceptHeaders {
		if mediaType == schema2.MediaTypeManifest {
			return true
		}
	}
	return false
}

// Put creates or updates the named manifest.
func (r *repository) Put(ctx context.Context, manifest distribution.Manifest, options ...distribution.ManifestServiceOption) (digest.Digest, error) {
	var canonical []byte

	// Resolve the payload in the manifest.
	mediatype, payload, err := manifest.Payload()
	if err != nil {
		return digest.Digest(""), err
	}

	switch manifest.(type) {
	case *schema1.SignedManifest:
		canonical = manifest.(*schema1.SignedManifest).Canonical
	case *schema2.DeserializedManifest:
		canonical = payload
	default:
		err := fmt.Errorf("unrecognized manifest type %T", manifest)
		return "", regapi.ErrorCodeManifestInvalid.WithDetail(err)
	}

	if !r.acceptschema2 {
		if _, ok := manifest.(*schema1.SignedManifest); !ok {
			err := fmt.Errorf("schema version 2 disabled")
			return "", regapi.ErrorCodeManifestInvalid.WithDetail(err)
		}
	}

	// Calculate digest
	dgst := digest.FromBytes(canonical)

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

	if err := r.registryOSClient.ImageStreamMappings(r.namespace).Create(&ism); err != nil {
		// if the error was that the image stream wasn't found, try to auto provision it
		statusErr, ok := err.(*kerrors.StatusError)
		if !ok {
			context.GetLogger(r.ctx).Errorf("error creating ImageStreamMapping: %s", err)
			return digest.Digest(""), err
		}

		if quotautil.IsErrorQuotaExceeded(statusErr) {
			context.GetLogger(r.ctx).Errorf("denied creating ImageStreamMapping: %v", statusErr)
			return digest.Digest(""), distribution.ErrAccessDenied
		}

		status := statusErr.ErrStatus
		if status.Code != http.StatusNotFound || (strings.ToLower(status.Details.Kind) != "imagestream" /*pre-1.2*/ && strings.ToLower(status.Details.Kind) != "imagestreams") || status.Details.Name != r.name {
			context.GetLogger(r.ctx).Errorf("error creating ImageStreamMapping: %s", err)
			return digest.Digest(""), err
		}

		stream := imageapi.ImageStream{
			ObjectMeta: kapi.ObjectMeta{
				Name: r.name,
			},
		}

		client, ok := UserClientFrom(r.ctx)
		if !ok {
			context.GetLogger(r.ctx).Errorf("error creating user client to auto provision image stream: Origin user client unavailable")
			return digest.Digest(""), statusErr
		}

		if _, err := client.ImageStreams(r.namespace).Create(&stream); err != nil {
			if quotautil.IsErrorQuotaExceeded(err) {
				context.GetLogger(r.ctx).Errorf("denied creating ImageStream: %v", err)
				return digest.Digest(""), distribution.ErrAccessDenied
			}
			context.GetLogger(r.ctx).Errorf("error auto provisioning ImageStream: %s", err)
			return digest.Digest(""), statusErr
		}

		// try to create the ISM again
		if err := r.registryOSClient.ImageStreamMappings(r.namespace).Create(&ism); err != nil {
			if quotautil.IsErrorQuotaExceeded(err) {
				context.GetLogger(r.ctx).Errorf("denied a creation of ImageStreamMapping: %v", err)
				return digest.Digest(""), distribution.ErrAccessDenied
			}
			context.GetLogger(r.ctx).Errorf("error creating ImageStreamMapping: %s", err)
			return digest.Digest(""), err
		}
	}

	return dgst, nil
}

// fillImageWithMetadata fills a given image with metadata. It also corrects layer sizes with blob sizes. Newer
// Docker client versions don't set layer sizes in the manifest at all. Origin master needs correct layer
// sizes for proper image quota support. That's why we need to fill the metadata in the registry.
func (r *repository) fillImageWithMetadata(manifest distribution.Manifest, image *imageapi.Image) error {
	if deserializedManifest, ok := manifest.(*schema2.DeserializedManifest); ok {
		configBytes, err := r.Blobs(r.ctx).Get(r.ctx, deserializedManifest.Config.Digest)
		if err != nil {
			context.GetLogger(r.ctx).Errorf("failed to get image config %s: %v", deserializedManifest.Config.Digest.String(), err)
			return err
		}
		image.DockerImageConfig = string(configBytes)
	}

	if signedManifest, ok := manifest.(*schema1.SignedManifest); ok {
		signatures, err := signedManifest.Signatures()
		if err != nil {
			return err
		}

		for _, signDigest := range signatures {
			image.DockerImageSignatures = append(image.DockerImageSignatures, signDigest)
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
			context.GetLogger(r.ctx).Errorf("failed to stat blobs %s of image %s", layer.Name, image.DockerImageReference)
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
	context.GetLogger(r.ctx).Infof("total size of image %s with docker ref %s: %d", image.Name, image.DockerImageReference, size)

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
	secrets, err := r.registryOSClient.ImageStreamSecrets(r.namespace).Secrets(r.name, kapi.ListOptions{})
	if err != nil {
		context.GetLogger(r.ctx).Errorf("error getting secrets for repository %q: %v", r.Named().Name(), err)
		secrets = &kapi.SecretList{}
	}
	credentials := importer.NewCredentialsForSecrets(secrets.Items)
	return importer.NewContext(secureTransport, insecureTransport).WithCredentials(credentials)
}

// getImageStream retrieves the ImageStream for r.
func (r *repository) getImageStream() (*imageapi.ImageStream, error) {
	return r.registryOSClient.ImageStreams(r.namespace).Get(r.name)
}

// getImage retrieves the Image with digest `dgst`.
func (r *repository) getImage(dgst digest.Digest) (*imageapi.Image, error) {
	return r.registryOSClient.Images().Get(dgst.String())
}

// getImageStreamImage retrieves the Image with digest `dgst` for the ImageStream
// associated with r. This ensures the image belongs to the image stream.
func (r *repository) getImageStreamImage(dgst digest.Digest) (*imageapi.ImageStreamImage, error) {
	return r.registryOSClient.ImageStreamImages(r.namespace).Get(r.name, dgst.String())
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
	if supportsSchema2 {
		if image.DockerImageManifestMediaType == schema2.MediaTypeManifest {
			manifest, err = r.deserializedManifestFromImage(image)
		} else {
			manifest, err = r.signedManifestFromImage(image)
		}
	} else {
		if image.DockerImageManifestMediaType == schema2.MediaTypeManifest {
			err = fmt.Errorf("unable to convert new image to old one")
			err = regapi.ErrorCodeManifestInvalid.WithDetail(err)
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
		return nil, fmt.Errorf("unable to convert new image to old one")
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
	repository, err := dockerRegistry.Repository(r.ctx, r.Named())
	if err != nil {
		return nil, err
	}

	manifestService, err := repository.Manifests(r.ctx)
	if err != nil {
		return nil, err
	}

	signaturesGetter, ok := manifestService.(distribution.SignaturesGetter)
	if !ok {
		return nil, fmt.Errorf("unable to convert ManifestService into SignaturesGetter")
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
