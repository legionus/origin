package server

import (
	"fmt"
	"path"
	"strings"

	"github.com/docker/distribution/digest"
)

const (
	storagePathVersion = "v1"                   // fixed storage layout version
	storagePathRoot    = "/openshift/registry/" // all driver paths have a prefix
	uploadsDirectory   = "_uploads"

	multilevelHexPrefixLength = 2
)

// blobAlgorithmReplacer does some very simple path sanitization for user
// input. Mostly, this is to provide some hierarchy for tarsum digests. Paths
// should be "safe" before getting this far due to strict digest requirements
// but we can add further path conversion here, if needed.
var blobAlgorithmReplacer = strings.NewReplacer(
	"+", "/",
	".", "/",
	";", "/",
)

func rootPath() string {
	return path.Join(storagePathRoot, storagePathVersion)
}

// Uploads:
//
// uploadDataPathSpec:      <root>/v1/repositories/<name>/_uploads/<id>/data
// uploadStartedAtPathSpec: <root>/v1/repositories/<name>/_uploads/<id>/startedat
// uploadHashStatePathSpec: <root>/v1/repositories/<name>/_uploads/<id>/hashstates/<algorithm>/<offset>
//
// Blob Store:
//
// blobPathSpec:            <root>/v1/blobs/<algorithm>/<first two hex bytes of digest>/<hex digest>
// blobDataPathSpec:        <root>/v1/blobs/<algorithm>/<first two hex bytes of digest>/<hex digest>/data
// blobMediaTypePathSpec:   <root>/v1/blobs/<algorithm>/<first two hex bytes of digest>/<hex digest>/data
//
// For more information on the semantic meaning of each path and their
// contents, please see the path spec documentation.
func pathFor(spec pathSpec) (string, error) {

	// Switch on the path object type and return the appropriate path. At
	// first glance, one may wonder why we don't use an interface to
	// accomplish this. By keep the formatting separate from the pathSpec, we
	// keep separate the path generation componentized. These specs could be
	// passed to a completely different mapper implementation and generate a
	// different set of paths.
	//
	// For example, imagine migrating from one backend to the other: one could
	// build a filesystem walker that converts a string path in one version,
	// to an intermediate path object, than can be consumed and mapped by the
	// other version.

	rootPrefix := []string{storagePathRoot, storagePathVersion}
	repoPrefix := append(rootPrefix, "repositories")

	switch v := spec.(type) {
	case blobPathSpec:
		components, err := digestPathComponents(v.digest, true)
		if err != nil {
			return "", err
		}

		blobPathPrefix := append(rootPrefix, "blobs")

		return path.Join(append(blobPathPrefix, components...)...), nil
	case blobDataPathSpec:
		blobPathPrefix, err := pathFor(blobPathSpec{
			digest: v.digest,
		})
		if err != nil {
			return "", err
		}

		return path.Join(blobPathPrefix, "data"), nil
	case uploadDataPathSpec:
		return path.Join(append(repoPrefix, v.name, uploadsDirectory, v.id, "data")...), nil
	case uploadStartedAtPathSpec:
		return path.Join(append(repoPrefix, v.name, uploadsDirectory, v.id, "startedat")...), nil
	case uploadHashStatePathSpec:
		offset := fmt.Sprintf("%d", v.offset)
		if v.list {
			offset = "" // Limit to the prefix for listing offsets.
		}
		return path.Join(append(repoPrefix, v.name, uploadsDirectory, v.id, "hashstates", string(v.alg), offset)...), nil
	case repositoriesRootPathSpec:
		return path.Join(repoPrefix...), nil
	default:
		// TODO(sday): This is an internal error. Ensure it doesn't escape (panic?).
		return "", fmt.Errorf("unknown path spec: %#v", v)
	}
}

// pathSpec is a type to mark structs as path specs. There is no
// implementation because we'd like to keep the specs and the mappers
// decoupled.
type pathSpec interface {
	pathSpec()
}

// blobPathSpec contains the path for the registry global blob store.
type blobPathSpec struct {
	digest digest.Digest
}

func (blobPathSpec) pathSpec() {}

// blobDataPathSpec contains the path for the registry global blob store. For
// now, this contains layer data, exclusively.
type blobDataPathSpec struct {
	digest digest.Digest
}

func (blobDataPathSpec) pathSpec() {}

// uploadDataPathSpec defines the path parameters of the data file for
// uploads.
type uploadDataPathSpec struct {
	name string
	id   string
}

func (uploadDataPathSpec) pathSpec() {}

// uploadDataPathSpec defines the path parameters for the file that stores the
// start time of an uploads. If it is missing, the upload is considered
// unknown. Admittedly, the presence of this file is an ugly hack to make sure
// we have a way to cleanup old or stalled uploads that doesn't rely on driver
// FileInfo behavior. If we come up with a more clever way to do this, we
// should remove this file immediately and rely on the startetAt field from
// the client to enforce time out policies.
type uploadStartedAtPathSpec struct {
	name string
	id   string
}

func (uploadStartedAtPathSpec) pathSpec() {}

// uploadHashStatePathSpec defines the path parameters for the file that stores
// the hash function state of an upload at a specific byte offset. If `list` is
// set, then the path mapper will generate a list prefix for all hash state
// offsets for the upload identified by the name, id, and alg.
type uploadHashStatePathSpec struct {
	name   string
	id     string
	alg    digest.Algorithm
	offset int64
	list   bool
}

func (uploadHashStatePathSpec) pathSpec() {}

// repositoriesRootPathSpec returns the root of repositories
type repositoriesRootPathSpec struct {
}

func (repositoriesRootPathSpec) pathSpec() {}

// digestPathComponents provides a consistent path breakdown for a given
// digest. For a generic digest, it will be as follows:
//
// 	<algorithm>/<hex digest>
//
// Most importantly, for tarsum, the layout looks like this:
//
// 	tarsum/<version>/<digest algorithm>/<full digest>
//
// If multilevel is true, the first two bytes of the digest will separate
// groups of digest folder. It will be as follows:
//
// 	<algorithm>/<first two bytes of digest>/<full digest>
//
func digestPathComponents(dgst digest.Digest, multilevel bool) ([]string, error) {
	if err := dgst.Validate(); err != nil {
		return nil, err
	}

	algorithm := blobAlgorithmReplacer.Replace(string(dgst.Algorithm()))
	hex := dgst.Hex()
	prefix := []string{algorithm}

	var suffix []string

	if multilevel {
		suffix = append(suffix, hex[:multilevelHexPrefixLength])
	}

	suffix = append(suffix, hex)

	if tsi, err := digest.ParseTarSum(dgst.String()); err == nil {
		// We have a tarsum!
		version := tsi.Version
		if version == "" {
			version = "v0"
		}

		prefix = []string{
			"tarsum",
			version,
			tsi.Algorithm,
		}
	}

	return append(prefix, suffix...), nil
}
