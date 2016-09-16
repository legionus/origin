package server

import (
	"errors"
	"sync"
	"time"

	"github.com/docker/distribution"
	"github.com/docker/distribution/digest"

	imageapi "github.com/openshift/origin/pkg/image/api"
)

const (
	BlobIndexContextVar = "openshift.registry.blobindex"
)

var (
	ErrBlobIndexNotFound      = errors.New("blob not found")
	ErrBlobIndexAlreadyExists = errors.New("blob already exists")
)

type BlobReference struct {
	Digest     digest.Digest
	Reference  imageapi.DockerImageReference
	Descriptor distribution.Descriptor
}

type blobInfo struct {
	Reference  map[imageapi.DockerImageReference]time.Time
	Descriptor distribution.Descriptor
}

func NewBlobIndex() *BlobIndex {
	return &BlobIndex{
		index: make(map[digest.Digest]*blobInfo),
	}
}

type BlobIndex struct {
	mu    sync.RWMutex
	index map[digest.Digest]*blobInfo
}

func (i *BlobIndex) Add(ref *BlobReference) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if _, ok := i.index[ref.Digest]; !ok {
		i.index[ref.Digest] = &blobInfo{
			Reference:  map[imageapi.DockerImageReference]time.Time{ref.Reference: time.Now()},
			Descriptor: ref.Descriptor,
		}
		return nil
	}

	if _, ok := i.index[ref.Digest].Reference[ref.Reference]; !ok {
		i.index[ref.Digest].Reference[ref.Reference] = time.Now()
		return nil
	}

	return ErrBlobIndexAlreadyExists
}

func (i *BlobIndex) Remove(ref *BlobReference) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if _, ok := i.index[ref.Digest]; !ok {
		return ErrBlobIndexNotFound
	}

	if _, ok := i.index[ref.Digest].Reference[ref.Reference]; !ok {
		return ErrBlobIndexNotFound
	}

	if len(i.index[ref.Digest].Reference) == 1 {
		delete(i.index, ref.Digest)
		return nil
	}

	delete(i.index[ref.Digest].Reference, ref.Reference)
	return nil
}

func (i *BlobIndex) GetDescriptor(dgst digest.Digest) (distribution.Descriptor, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	ref, ok := i.index[dgst]
	if !ok {
		return distribution.Descriptor{}, ErrBlobIndexNotFound
	}
	return ref.Descriptor, nil
}

func (i *BlobIndex) Get(ref *BlobReference) (distribution.Descriptor, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	info, ok := i.index[ref.Digest]
	if !ok {
		return distribution.Descriptor{}, ErrBlobIndexNotFound
	}

	if _, ok := info.Reference[ref.Reference]; !ok {
		return distribution.Descriptor{}, ErrBlobIndexNotFound
	}

	return info.Descriptor, nil
}
