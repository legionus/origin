package server

import (
	"net/http"

	"github.com/docker/distribution"
	"github.com/docker/distribution/context"
	"github.com/docker/distribution/digest"

	"github.com/openshift/origin/pkg/dockerregistry/server/audit"
	"github.com/openshift/origin/pkg/dockerregistry/server/metrics"
)

// auditBlobStore wraps a distribution.BlobStore to track operation result and
// write it in the audit log.
type auditBlobStore struct {
	store distribution.BlobStore
	repo  *repository
}

var _ distribution.BlobStore = &auditBlobStore{}

func (b *auditBlobStore) Stat(ctx context.Context, dgst digest.Digest) (distribution.Descriptor, error) {
	defer metrics.NewTimer(metrics.RequestDurationSummary, []string{"blobstore.stat", b.repo.Named().Name()}).Stop()

	audit.GetLogger(ctx).Log("BlobStore.Stat")
	desc, err := b.store.Stat(ctx, dgst)
	audit.GetLogger(ctx).LogResult(err, "BlobStore.Stat")
	return desc, err
}

func (b *auditBlobStore) Get(ctx context.Context, dgst digest.Digest) ([]byte, error) {
	defer metrics.NewTimer(metrics.RequestDurationSummary, []string{"blobstore.get", b.repo.Named().Name()}).Stop()

	audit.GetLogger(ctx).Log("BlobStore.Get")
	blob, err := b.store.Get(ctx, dgst)
	audit.GetLogger(ctx).LogResult(err, "BlobStore.Get")
	return blob, err
}

func (b *auditBlobStore) Open(ctx context.Context, dgst digest.Digest) (distribution.ReadSeekCloser, error) {
	defer metrics.NewTimer(metrics.RequestDurationSummary, []string{"blobstore.open", b.repo.Named().Name()}).Stop()

	audit.GetLogger(ctx).Log("BlobStore.Open")
	reader, err := b.store.Open(ctx, dgst)
	audit.GetLogger(ctx).LogResult(err, "BlobStore.Open")
	return reader, err
}

func (b *auditBlobStore) Put(ctx context.Context, mediaType string, p []byte) (distribution.Descriptor, error) {
	defer metrics.NewTimer(metrics.RequestDurationSummary, []string{"blobstore.put", b.repo.Named().Name()}).Stop()

	audit.GetLogger(ctx).Log("BlobStore.Put")
	desc, err := b.store.Put(ctx, mediaType, p)
	audit.GetLogger(ctx).LogResult(err, "BlobStore.Put")
	return desc, err
}

func (b *auditBlobStore) Create(ctx context.Context, options ...distribution.BlobCreateOption) (distribution.BlobWriter, error) {
	defer metrics.NewTimer(metrics.RequestDurationSummary, []string{"blobstore.create", b.repo.Named().Name()}).Stop()

	audit.GetLogger(ctx).Log("BlobStore.Create")
	writer, err := b.store.Create(ctx, options...)
	audit.GetLogger(ctx).LogResult(err, "BlobStore.Create")
	return &blobWriter{
		BlobWriter: writer,
		repo:       b.repo,
	}, err
}

func (b *auditBlobStore) Resume(ctx context.Context, id string) (distribution.BlobWriter, error) {
	defer metrics.NewTimer(metrics.RequestDurationSummary, []string{"blobstore.resume", b.repo.Named().Name()}).Stop()

	audit.GetLogger(ctx).Log("BlobStore.Resume")
	writer, err := b.store.Resume(ctx, id)
	audit.GetLogger(ctx).LogResult(err, "BlobStore.Resume")
	return writer, err
}

func (b *auditBlobStore) ServeBlob(ctx context.Context, w http.ResponseWriter, req *http.Request, dgst digest.Digest) error {
	defer metrics.NewTimer(metrics.RequestDurationSummary, []string{"blobstore.serveblob", b.repo.Named().Name()}).Stop()

	audit.GetLogger(ctx).Log("BlobStore.ServeBlob")
	err := b.store.ServeBlob(ctx, w, req, dgst)
	audit.GetLogger(ctx).LogResult(err, "BlobStore.ServeBlob")
	return err
}

func (b *auditBlobStore) Delete(ctx context.Context, dgst digest.Digest) error {
	defer metrics.NewTimer(metrics.RequestDurationSummary, []string{"blobstore.delete", b.repo.Named().Name()}).Stop()

	audit.GetLogger(ctx).Log("BlobStore.Delete")
	err := b.store.Delete(ctx, dgst)
	audit.GetLogger(ctx).LogResult(err, "BlobStore.Delete")
	return err
}

type blobWriter struct {
	distribution.BlobWriter

	repo *repository
}

func (bw *blobWriter) Commit(ctx context.Context, provisional distribution.Descriptor) (canonical distribution.Descriptor, err error) {
	defer metrics.NewTimer(metrics.RequestDurationSummary, []string{"blobwriter.commit", bw.repo.Named().Name()}).Stop()

	audit.GetLogger(ctx).Log("BlobWriter.Commit")
	desc, err := bw.BlobWriter.Commit(ctx, provisional)
	audit.GetLogger(ctx).LogResult(err, "BlobWriter.Commit")
	return desc, err
}

func (bw *blobWriter) Cancel(ctx context.Context) error {
	defer metrics.NewTimer(metrics.RequestDurationSummary, []string{"blobwriter.cancel", bw.repo.Named().Name()}).Stop()

	audit.GetLogger(ctx).Log("BlobWriter.Cancel")
	err := bw.BlobWriter.Cancel(ctx)
	audit.GetLogger(ctx).LogResult(err, "BlobWriter.Cancel")
	return err
}
