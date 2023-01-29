package s3

import (
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	as3 "github.com/fclairamb/afero-s3"
	"github.com/goxiaoy/vfs"
	"net/url"
	"path"
	"strings"
	"time"
)

type Blob struct {
	vfs.FS

	session           *session.Session
	bucket            string
	publicAccessUrl   url.URL
	internalAccessUrl url.URL
	defaultExpire     time.Duration

	s3Api *s3.S3
}

var _ vfs.Blob = (*Blob)(nil)

func NewBlob(session *session.Session, bucket string, publicAccessUrl url.URL, internalAccessUrl url.URL, defaultExpire time.Duration) *Blob {
	// Initialize the file system
	s3Fs := as3.NewFs(bucket, session)
	s3Api := s3.New(session)
	return &Blob{
		FS:                s3Fs,
		session:           session,
		bucket:            bucket,
		publicAccessUrl:   publicAccessUrl,
		internalAccessUrl: internalAccessUrl,
		defaultExpire:     defaultExpire,
		s3Api:             s3Api,
	}
}

func (b *Blob) PreSignedURL(ctx context.Context, name string, args ...vfs.LinkOptions) (res *vfs.Link, err error) {
	r, _ := b.s3Api.PutObjectRequest(&s3.PutObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(strings.TrimPrefix(name, "/")),
	})
	t := b.defaultExpire
	if len(args) > 0 && args[0].Expire != nil {
		t = *args[0].Expire
	}
	res.URL, err = r.Presign(t)
	return
}

func (b *Blob) PublicUrl(ctx context.Context, name string) (res *vfs.Link, err error) {
	url := b.publicAccessUrl
	url.Path = path.Join(url.Path, name)
	res.URL = url.String()
	return
}

func (b *Blob) InternalUrl(ctx context.Context, name string, args ...vfs.LinkOptions) (res *vfs.Link, err error) {
	url := b.internalAccessUrl
	url.Path = path.Join(url.Path, name)
	res.URL = url.String()
	return
}
