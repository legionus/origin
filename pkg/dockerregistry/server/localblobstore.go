package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/docker/distribution"
	"github.com/docker/distribution/context"
	"github.com/docker/distribution/digest"
	"github.com/docker/distribution/registry/storage/driver"
	"github.com/docker/distribution/uuid"
)

const (
	// TODO(stevvooe): This should configurable in the future.
	blobCacheControlMaxAge = 365 * 24 * time.Hour
)

type localBlobStore struct {
	distribution.BlobStore

	ctx      context.Context
	driver   driver.StorageDriver
	repo     *repository
	redirect bool // allows disabling URLFor redirects

}

var _ distribution.BlobStore = &localBlobStore{}

func (bs *localBlobStore) blobPath(dgst digest.Digest) (string, error) {
	bp, err := pathFor(blobDataPathSpec{
		digest: dgst,
	})

	if err != nil {
		return "", err
	}

	return bp, nil
}

func (bs *localBlobStore) Stat(ctx context.Context, dgst digest.Digest) (distribution.Descriptor, error) {
	path, err := pathFor(blobDataPathSpec{
		digest: dgst,
	})

	if err != nil {
		return distribution.Descriptor{}, err
	}

	fi, err := bs.driver.Stat(ctx, path)
	if err != nil {
		switch err := err.(type) {
		case driver.PathNotFoundError:
			// XXXlegion: we did not find digest it in our layout, but it can be in the old layout.
			return bs.BlobStore.Stat(ctx, dgst)
		default:
			return distribution.Descriptor{}, err
		}
	}

	if fi.IsDir() {
		context.GetLogger(ctx).Errorf("blob path should not be a directory: %q", path)
		return distribution.Descriptor{}, distribution.ErrBlobUnknown
	}

	return distribution.Descriptor{
		Size:      fi.Size(),
		MediaType: "application/octet-stream",
		Digest:    dgst,
	}, nil
}

func (bs *localBlobStore) Put(ctx context.Context, mediaType string, p []byte) (distribution.Descriptor, error) {
	context.GetLogger(ctx).Debug("(*localBlobStore).Put: starting")

	dgst, err := digest.FromBytes(p)
	if err != nil {
		context.GetLogger(ctx).Errorf("(*localBlobStore).Put: error digesting content: %v, %s", err, string(p))
		return distribution.Descriptor{}, err
	}

	desc, err := bs.Stat(ctx, dgst)
	if err == nil {
		return desc, nil
	} else if err != distribution.ErrBlobUnknown {
		context.GetLogger(ctx).Errorf("(*localBlobStore).Put: error stating content (%v): %#v", dgst, err)
		return distribution.Descriptor{}, err
	}

	bp, err := bs.blobPath(dgst)
	if err != nil {
		return distribution.Descriptor{}, err
	}

	desc = distribution.Descriptor{
		Size:      int64(len(p)),
		MediaType: "application/octet-stream",
		Digest:    dgst,
	}
	err = bs.driver.PutContent(ctx, bp, p)

	if err == nil {
		context.GetLogger(ctx).Debugf("(*localBlobStore).Put: name=%s digest=%s", bs.repo.Name(), desc.Digest)
	}
	return desc, err
}

func (bs *localBlobStore) Get(ctx context.Context, dgst digest.Digest) ([]byte, error) {
	bp, err := bs.blobPath(dgst)
	if err != nil {
		return nil, err
	}

	p, err := bs.driver.GetContent(ctx, bp)
	if err != nil {
		switch err.(type) {
		case driver.PathNotFoundError:
			// XXXlegion: we did not find digest it in our layout, but it can be in the old layout.
			return bs.BlobStore.Get(ctx, dgst)
		}

		return nil, err
	}

	return p, err
}

func (bs *localBlobStore) Open(ctx context.Context, dgst digest.Digest) (distribution.ReadSeekCloser, error) {
	desc, err := bs.Stat(ctx, dgst)
	if err != nil {
		switch err.(type) {
		case driver.PathNotFoundError:
			// XXXlegion: we did not find digest it in our layout, but it can be in the old layout.
			return bs.BlobStore.Open(ctx, dgst)
		}
		return nil, err
	}

	path, err := bs.blobPath(desc.Digest)
	if err != nil {
		return nil, err
	}

	return newFileReader(ctx, bs.driver, path, desc.Size)
}

func (bs *localBlobStore) ServeBlob(ctx context.Context, w http.ResponseWriter, r *http.Request, dgst digest.Digest) error {
	desc, err := bs.Stat(ctx, dgst) // access check
	if err != nil {
		switch err.(type) {
		case driver.PathNotFoundError:
			// XXXlegion: we did not find digest it in our layout, but it can be in the old layout.
			return bs.BlobStore.ServeBlob(ctx, w, r, dgst)
		}
		return err
	}

	if desc.MediaType != "" {
		// Set the repository local content type.
		w.Header().Set("Content-Type", desc.MediaType)
	}

	path, err := bs.blobPath(desc.Digest)
	if err != nil {
		return err
	}

	redirectURL, err := bs.driver.URLFor(ctx, path, map[string]interface{}{"method": r.Method})

	switch err.(type) {
	case nil:
		if bs.redirect {
			// Redirect to storage URL.
			http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
			return err
		}

	case driver.ErrUnsupportedMethod:
		// Fallback to serving the content directly.
		br, err := newFileReader(ctx, bs.driver, path, desc.Size)
		if err != nil {
			return err
		}
		defer br.Close()

		w.Header().Set("ETag", fmt.Sprintf(`"%s"`, desc.Digest)) // If-None-Match handled by ServeContent
		w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%.f", blobCacheControlMaxAge.Seconds()))

		if w.Header().Get("Docker-Content-Digest") == "" {
			w.Header().Set("Docker-Content-Digest", desc.Digest.String())
		}

		if w.Header().Get("Content-Type") == "" {
			// Set the content type if not already set.
			w.Header().Set("Content-Type", desc.MediaType)
		}

		if w.Header().Get("Content-Length") == "" {
			// Set the content length if not already set.
			w.Header().Set("Content-Length", fmt.Sprint(desc.Size))
		}

		http.ServeContent(w, r, desc.Digest.String(), time.Time{}, br)
		return nil
	}

	// Some unexpected error.
	return err
}

func (bs *localBlobStore) Create(ctx context.Context) (distribution.BlobWriter, error) {
	context.GetLogger(ctx).Debug("(*localBlobStore).Create: starting")

	uuid := uuid.Generate().String()
	startedAt := time.Now().UTC()

	path, err := pathFor(uploadDataPathSpec{
		name: bs.repo.Name(),
		id:   uuid,
	})

	if err != nil {
		return nil, err
	}

	startedAtPath, err := pathFor(uploadStartedAtPathSpec{
		name: bs.repo.Name(),
		id:   uuid,
	})

	if err != nil {
		return nil, err
	}

	// Write a startedat file for this upload
	if err := bs.driver.PutContent(ctx, startedAtPath, []byte(startedAt.Format(time.RFC3339))); err != nil {
		return nil, err
	}

	return bs.newBlobUpload(ctx, uuid, path, startedAt)
}

func (bs *localBlobStore) Delete(ctx context.Context, dgst digest.Digest) error {
	context.GetLogger(ctx).Debug("(*localBlobStore).Delete starting")
	_, err := bs.Stat(ctx, dgst)
	if err != nil {
		switch err.(type) {
		case driver.PathNotFoundError:
			// XXXlegion: we did not find digest it in our layout, but it can be in the old layout.
			return bs.BlobStore.Delete(ctx, dgst)
		}
		return err
	}
	context.GetLogger(ctx).Warn("(*localBlobStore).Delete not implemented")
	return nil
}

func (bs *localBlobStore) Resume(ctx context.Context, id string) (distribution.BlobWriter, error) {
	context.GetLogger(ctx).Debug("(*localBlobStore).Resume: starting")

	startedAtPath, err := pathFor(uploadStartedAtPathSpec{
		name: bs.repo.Name(),
		id:   id,
	})

	if err != nil {
		return nil, err
	}

	startedAtBytes, err := bs.driver.GetContent(ctx, startedAtPath)
	if err != nil {
		switch err := err.(type) {
		case driver.PathNotFoundError:
			return nil, distribution.ErrBlobUploadUnknown
		default:
			return nil, err
		}
	}

	startedAt, err := time.Parse(time.RFC3339, string(startedAtBytes))
	if err != nil {
		return nil, err
	}

	path, err := pathFor(uploadDataPathSpec{
		name: bs.repo.Name(),
		id:   id,
	})

	if err != nil {
		return nil, err
	}

	return bs.newBlobUpload(ctx, id, path, startedAt)
}

// newBlobUpload allocates a new upload controller with the given state.
func (bs *localBlobStore) newBlobUpload(ctx context.Context, uuid, path string, startedAt time.Time) (distribution.BlobWriter, error) {
	fw, err := newFileWriter(ctx, bs.driver, path)
	if err != nil {
		return nil, err
	}
	return &blobWriter{
		blobStore:              bs,
		id:                     uuid,
		startedAt:              startedAt,
		digester:               digest.Canonical.New(),
		bufferedFileWriter:     *fw,
		resumableDigestEnabled: true,
	}, nil
}
