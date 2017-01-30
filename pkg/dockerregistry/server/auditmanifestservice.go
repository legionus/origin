package server

import (
	"github.com/docker/distribution"
	"github.com/docker/distribution/context"
	"github.com/docker/distribution/digest"

	"github.com/openshift/origin/pkg/dockerregistry/server/audit"
	"github.com/openshift/origin/pkg/dockerregistry/server/metrics"
)

// auditManifestService wraps a distribution.ManifestService to track operation result and
// write it in the audit log.
type auditManifestService struct {
	manifests distribution.ManifestService
	repo      *repository
}

var _ distribution.ManifestService = &auditManifestService{}

func (m *auditManifestService) Exists(ctx context.Context, dgst digest.Digest) (bool, error) {
	defer metrics.NewTimer(metrics.RequestDurationSummary, []string{"manifestservice.exists", m.repo.Named().Name()}).Stop()

	audit.GetLogger(ctx).Log("ManifestService.Exists")
	exists, err := m.manifests.Exists(ctx, dgst)
	audit.GetLogger(ctx).LogResult(err, "ManifestService.Exists")
	return exists, err
}

func (m *auditManifestService) Get(ctx context.Context, dgst digest.Digest, options ...distribution.ManifestServiceOption) (distribution.Manifest, error) {
	defer metrics.NewTimer(metrics.RequestDurationSummary, []string{"manifestservice.get", m.repo.Named().Name()}).Stop()

	audit.GetLogger(ctx).Log("ManifestService.Get")
	manifest, err := m.manifests.Get(ctx, dgst, options...)
	audit.GetLogger(ctx).LogResult(err, "ManifestService.Get")
	return manifest, err
}

func (m *auditManifestService) Put(ctx context.Context, manifest distribution.Manifest, options ...distribution.ManifestServiceOption) (digest.Digest, error) {
	defer metrics.NewTimer(metrics.RequestDurationSummary, []string{"manifestservice.put", m.repo.Named().Name()}).Stop()

	audit.GetLogger(ctx).Log("ManifestService.Put")
	dgst, err := m.manifests.Put(ctx, manifest, options...)
	audit.GetLogger(ctx).LogResult(err, "ManifestService.Put")
	return dgst, err
}

func (m *auditManifestService) Delete(ctx context.Context, dgst digest.Digest) error {
	defer metrics.NewTimer(metrics.RequestDurationSummary, []string{"manifestservice.delete", m.repo.Named().Name()}).Stop()

	audit.GetLogger(ctx).Log("ManifestService.Delete")
	err := m.manifests.Delete(ctx, dgst)
	audit.GetLogger(ctx).LogResult(err, "ManifestService.Delete")
	return err
}
