package chaindigest

import (
	"io"
	"strings"

	"github.com/docker/distribution/digest"
)

const ChainPrefix = "chain:"

type ChainDigest struct {
	digest.Digest
}

func (c ChainDigest) String() string {
	return ChainPrefix + c.Digest.String()
}

func (d ChainDigest) Validate() error {
	s := string(d.Digest)

	if !strings.HasPrefix(s, ChainPrefix) {
		return digest.ErrDigestInvalidFormat
	}

	return d.Digest.Validate()
}

func FromReader(rd io.Reader) (ChainDigest, error) {
	c := ChainDigest{}

	d, err := digest.FromReader(rd)
	if err != nil {
		return c, err
	}

	c.Digest = d

	return c, nil
}

func FromBytes(p []byte) ChainDigest {
	return ChainDigest{
		Digest: digest.FromBytes(p),
	}
}

func ParseChainDigest(s string) (ChainDigest, error) {
	c := ChainDigest{}

	if !strings.HasPrefix(s, ChainPrefix) {
		return c, digest.ErrDigestInvalidFormat
	}

	d, err := digest.ParseDigest(s[len(ChainPrefix):])
	if err != nil {
		return c, err
	}

	c.Digest = d

	return c, nil
}

func ParseDigest(s string) (ChainDigest, error) {
	c := ChainDigest{}

	d, err := digest.ParseDigest(s)
	if err != nil {
		return c, err
	}

	c.Digest = d

	return c, nil
}
