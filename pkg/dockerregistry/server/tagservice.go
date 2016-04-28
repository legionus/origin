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

	registryClient client.Interface
	imageStream    *imageapi.ImageStream
}

func (t tagService) Get(ctx context.Context, tag string) (distribution.Descriptor, error) {
	context.GetLogger(ctx).Debug("(tagService).Get: starting")

	te := imageapi.LatestTaggedImage(t.imageStream, tag)
	if te == nil {
		return distribution.Descriptor{}, distribution.ErrTagUnknown{Tag: tag}
	}
	dgst, err := digest.ParseDigest(te.Image)
	if err != nil {
		return distribution.Descriptor{}, err
	}

	return distribution.Descriptor{Digest: dgst}, nil
}

func (t tagService) All(ctx context.Context) ([]string, error) {
	context.GetLogger(ctx).Debug("(tagService).All: starting")

	tags := []string{}
	for tag := range t.imageStream.Status.Tags {
		tags = append(tags, tag)
	}
	return tags, nil
}

func (t tagService) Lookup(ctx context.Context, desc distribution.Descriptor) ([]string, error) {
	context.GetLogger(ctx).Debug("(tagService).Lookup: starting")

	tags := []string{}
	for tag, history := range t.imageStream.Status.Tags {
		if len(history.Items) == 0 {
			continue
		}

		dgst, err := digest.ParseDigest(history.Items[0].Image)
		if err != nil {
			return tags, err
		}

		if dgst == desc.Digest {
			tags = append(tags, tag)
		}
	}

	return tags, nil
}

func (t tagService) Tag(ctx context.Context, tag string, dgst distribution.Descriptor) error {
	context.GetLogger(ctx).Debug("(tagService).Tag: starting")

	image, err := t.registryClient.Images().Get(dgst.Digest.String())
	if err != nil {
		return err
	}

	ism := imageapi.ImageStreamMapping{
		ObjectMeta: kapi.ObjectMeta{
			Namespace: t.imageStream.Namespace,
			Name:      t.imageStream.Name,
		},
		Tag:   tag,
		Image: *image,
	}

	return t.registryClient.ImageStreamMappings(t.imageStream.Namespace).Create(&ism)
}

func (t tagService) Untag(ctx context.Context, tag string) error {
	context.GetLogger(ctx).Debug("(tagService).Untag: starting")
	return t.registryClient.ImageStreamTags(t.imageStream.Namespace).Delete(t.imageStream.Name, tag)
}
