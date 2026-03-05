package storage

import (
	"context"
	"io"
)

type Storage interface {
	Upload(ctx context.Context, filename string, size int64, r io.Reader) error
	Stat(ctx context.Context, path string) (*Metadata, error)
	Download(ctx context.Context, path string) (*Object, error)
	GetLink(ctx context.Context, path string) (string, error)
}

type Metadata struct {
	Size int64
}

type Object struct {
	io.ReadCloser
	*Metadata
}
