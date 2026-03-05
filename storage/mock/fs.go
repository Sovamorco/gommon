package mock

import (
	"bytes"
	"io"
	"io/fs"
	"time"
)

type fakeFs struct {
	objs map[string]object
}

type objectWrapper struct {
	io.ReadSeeker

	name    string
	size    int
	modTime time.Time
}

func (f *fakeFs) Open(name string) (fs.File, error) {
	obj, ok := f.objs[name]
	if !ok {
		return nil, fs.ErrNotExist
	}

	return &objectWrapper{
		ReadSeeker: bytes.NewReader(obj.data),
		name:       obj.filename,
		size:       len(obj.data),
		modTime:    time.Now(),
	}, nil
}

func (o *objectWrapper) Close() error {
	return nil
}

func (o *objectWrapper) Stat() (fs.FileInfo, error) {
	return o, nil
}

func (o *objectWrapper) Name() string {
	return o.name
}

func (o *objectWrapper) Size() int64 {
	return int64(o.size)
}

func (o *objectWrapper) Mode() fs.FileMode {
	//nolint:mnd // mock.
	return 0x777
}

func (o *objectWrapper) ModTime() time.Time {
	return o.modTime
}

func (o *objectWrapper) IsDir() bool {
	return false
}

//nolint:ireturn // required by interface.
func (o *objectWrapper) Sys() any {
	return nil
}
