package storage

import (
	"context"
	"io"
)

type Storage interface {
	Upload(ctx context.Context, filename string, size int64, uploader string, r io.Reader) (string, error)
	Stat(ctx context.Context, path string) (*Metadata, error)
	Download(ctx context.Context, path string) (*Object, error)
	GetLink(ctx context.Context, path string) (string, error)
}

type Metadata struct {
	Filename string
	Size     int64
	Uploader string
}

type Object struct {
	io.ReadCloser
	*Metadata
}
