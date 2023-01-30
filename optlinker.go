package vfs

import (
	"context"
	"net/url"
	"path"
)

type TokenValidator interface {
	Gen(ctx context.Context, key string, opts ...LinkOptions) (string, error)
	Validate(ctx context.Context, key string, opts ...LinkOptions) (bool, error)
}

// OptLinker wrap FS as Linker
type OptLinker struct {
	FS
	tv                TokenValidator
	publicAccessUrl   url.URL
	internalAccessUrl url.URL
}

func NewOptLinker(fs FS, publicAccessUrl url.URL, internalAccessUrl url.URL, tv TokenValidator) *OptLinker {
	return &OptLinker{
		FS:                fs,
		tv:                tv,
		publicAccessUrl:   publicAccessUrl,
		internalAccessUrl: internalAccessUrl,
	}
}

var _ Linker = (*OptLinker)(nil)

func (o *OptLinker) PreSignedURL(ctx context.Context, name string, args ...LinkOptions) (res *Link, err error) {
	token := ""
	if o.tv != nil {
		token, err = o.tv.Gen(ctx, name, args...)
		if err != nil {
			return nil, err
		}
	}
	url := o.publicAccessUrl
	url.Path = path.Join(url.Path, name)
	if len(token) > 0 {
		url.Query().Add("token", token)
	}
	res = &Link{}
	res.URL = url.String()
	return
}

func (o *OptLinker) PublicUrl(ctx context.Context, name string) (res *Link, err error) {
	url := o.publicAccessUrl
	url.Path = path.Join(url.Path, name)
	res = &Link{}
	res.URL = url.String()
	return
}

func (o *OptLinker) InternalUrl(ctx context.Context, name string, args ...LinkOptions) (res *Link, err error) {
	url := o.internalAccessUrl
	url.Path = path.Join(url.Path, name)
	res.URL = url.String()
	return
}
