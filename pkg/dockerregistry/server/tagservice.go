package server

import (
	"github.com/docker/distribution"
	"github.com/docker/distribution/context"
	"github.com/docker/distribution/digest"

	kapi "k8s.io/kubernetes/pkg/api"

	"github.com/openshift/origin/pkg/client"
	imageapi "github.com/openshift/origin/pkg/image/api"
)

type tagService struct {
	distribution.TagService

	repository     distribution.Repository
	registryClient client.Interface
	namespace      string
	name           string
	pullthrough    bool
}

func (t tagService) Get(ctx context.Context, tag string) (distribution.Descriptor, error) {
	imageStream, err := t.getImageStream()
	if err != nil {
		context.GetLogger(ctx).Errorf("Error retrieving ImageStream %s/%s: %v", t.namespace, t.name, err)
		return distribution.Descriptor{}, distribution.ErrRepositoryUnknown{Name: t.repository.Named().Name()}
	}

	te := imageapi.LatestTaggedImage(imageStream, tag)
	if te == nil {
		return distribution.Descriptor{}, distribution.ErrTagUnknown{Tag: tag}
	}
	dgst, err := digest.ParseDigest(te.Image)
	if err != nil {
		return distribution.Descriptor{}, err
	}

	if !t.pullthrough {
		image, err := t.getImage(dgst)
		if err != nil {
			return distribution.Descriptor{}, err
		}

		if !isImageManaged(image) {
			return distribution.Descriptor{}, distribution.ErrTagUnknown{Tag: tag}
		}
	}

	return distribution.Descriptor{Digest: dgst}, nil
}

func (t tagService) All(ctx context.Context) ([]string, error) {
	tags := []string{}

	imageStream, err := t.getImageStream()
	if err != nil {
		context.GetLogger(ctx).Errorf("Error retrieving ImageStream %s/%s: %v", t.namespace, t.name, err)
		return tags, distribution.ErrRepositoryUnknown{Name: t.repository.Named().Name()}
	}

	for tag, history := range imageStream.Status.Tags {
		if len(history.Items) == 0 {
			continue
		}
		if !t.pullthrough {
			dgst, err := digest.ParseDigest(history.Items[0].Image)
			if err != nil {
				return nil, err
			}

			image, err := t.getImage(dgst)
			if err != nil {
				return nil, err
			}

			if !isImageManaged(image) {
				continue
			}
		}
		tags = append(tags, tag)
	}
	return tags, nil
}

func (t tagService) Lookup(ctx context.Context, desc distribution.Descriptor) ([]string, error) {
	tags := []string{}

	imageStream, err := t.getImageStream()
	if err != nil {
		context.GetLogger(ctx).Errorf("Error retrieving ImageStream %s/%s: %v", t.namespace, t.name, err)
		return tags, distribution.ErrRepositoryUnknown{Name: t.repository.Named().Name()}
	}

	for tag, history := range imageStream.Status.Tags {
		if len(history.Items) == 0 {
			continue
		}

		dgst, err := digest.ParseDigest(history.Items[0].Image)
		if err != nil {
			return tags, err
		}

		if !t.pullthrough {
			image, err := t.getImage(dgst)
			if err != nil {
				return nil, err
			}

			if !isImageManaged(image) {
				continue
			}
		}

		if dgst == desc.Digest {
			tags = append(tags, tag)
		}
	}

	return tags, nil
}

func (t tagService) Tag(ctx context.Context, tag string, dgst distribution.Descriptor) error {
	imageStream, err := t.getImageStream()
	if err != nil {
		context.GetLogger(ctx).Errorf("Error retrieving ImageStream %s/%s: %v", t.namespace, t.name, err)
		return distribution.ErrRepositoryUnknown{Name: t.repository.Named().Name()}
	}

	image, err := t.registryClient.Images().Get(dgst.Digest.String())
	if err != nil {
		context.GetLogger(ctx).Errorf("(tagService).Tag: Unable to get image: %s", dgst.Digest.String())
		return err
	}
	image.SetResourceVersion("")

	if t.pullthrough && !isImageManaged(image) {
		return distribution.ErrRepositoryUnknown{Name: t.repository.Named().Name()}
	}

	ism := imageapi.ImageStreamMapping{
		ObjectMeta: kapi.ObjectMeta{
			Namespace: imageStream.Namespace,
			Name:      imageStream.Name,
		},
		Tag:   tag,
		Image: *image,
	}

	return t.registryClient.ImageStreamMappings(imageStream.Namespace).Create(&ism)
}

func (t tagService) Untag(ctx context.Context, tag string) error {
	imageStream, err := t.getImageStream()
	if err != nil {
		context.GetLogger(ctx).Errorf("Error retrieving ImageStream %s/%s: %v", t.namespace, t.name, err)
		return distribution.ErrRepositoryUnknown{Name: t.repository.Named().Name()}
	}

	te := imageapi.LatestTaggedImage(imageStream, tag)
	if te == nil {
		return distribution.ErrTagUnknown{Tag: tag}
	}

	if !t.pullthrough {
		dgst, err := digest.ParseDigest(te.Image)
		if err != nil {
			return err
		}

		image, err := t.getImage(dgst)
		if err != nil {
			return err
		}

		if !isImageManaged(image) {
			return distribution.ErrTagUnknown{Tag: tag}
		}
	}

	return t.registryClient.ImageStreamTags(imageStream.Namespace).Delete(imageStream.Name, tag)
}

// getImage retrieves the Image with digest `dgst`.
func (t *tagService) getImage(dgst digest.Digest) (*imageapi.Image, error) {
	return t.registryClient.Images().Get(dgst.String())
}

// getImageStream retrieves the ImageStream.
func (t *tagService) getImageStream() (*imageapi.ImageStream, error) {
	return t.registryClient.ImageStreams(t.namespace).Get(t.name)
}

// getImageStreamImage retrieves the Image with digest `dgst` for the ImageStream
// associated with r. This ensures the image belongs to the image stream.
func (t *tagService) getImageStreamImage(dgst digest.Digest) (*imageapi.ImageStreamImage, error) {
	return t.registryClient.ImageStreamImages(t.namespace).Get(t.name, dgst.String())
}

func isImageManaged(image *imageapi.Image) bool {
	managed, ok := image.ObjectMeta.Annotations[imageapi.ManagedByOpenShiftAnnotation]
	if !ok || managed != "true" {
		return false
	}
	return true
}
