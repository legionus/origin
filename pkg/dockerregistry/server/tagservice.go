package server

import (
	"github.com/docker/distribution"
	"github.com/docker/distribution/context"
	"github.com/docker/distribution/digest"

	imageapi "github.com/openshift/origin/pkg/image/api"
)

type tagService struct {
	distribution.TagService

	imageStream *imageapi.ImageStream
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

func (t tagService) Lookup(ctx context.Context, digest distribution.Descriptor) ([]string, error) {
	context.GetLogger(ctx).Debug("(tagService).Lookup: starting")
	return t.TagService.Lookup(ctx, digest)
}

// TODO(legion): Rewrite this method to store tag in etcd.
func (t tagService) Tag(ctx context.Context, tag string, digest distribution.Descriptor) error {
	context.GetLogger(ctx).Debug("(tagService).Tag: starting")
	return t.TagService.Tag(ctx, tag, digest)
}

func (t tagService) Untag(ctx context.Context, tag string) error {
	context.GetLogger(ctx).Debug("(tagService).Untag: starting")
	return t.TagService.Untag(ctx, tag)
}
