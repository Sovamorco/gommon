package mock

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/sovamorco/errorx"
	"github.com/sovamorco/gommon/storage"
)

//nolint:gochecknoinits // driver pattern.
func init() {
	storage.Register("mock", newMock)
}

type object struct {
	data     []byte
	filename string
	uploader string
}

type Storage struct {
	objs map[string]object
	port int
}

//nolint:ireturn // required by storage.Register.
func newMock(ctx context.Context, _ string) (storage.Storage, error) {
	var lc net.ListenConfig

	lis, err := lc.Listen(ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		return nil, errorx.Wrap(err, "get listener")
	}

	tlis, ok := lis.Addr().(*net.TCPAddr)
	if !ok {
		return nil, errorx.IllegalState.New("tcp listener addr not of *TCPAddr type, should never happen")
	}

	s := &Storage{
		objs: make(map[string]object),
		port: tlis.Port,
	}

	go func() {
		//nolint:gosec // used only in development.
		err := http.Serve(lis, http.FileServerFS(&fakeFs{objs: s.objs}))
		if err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Msg("Failed to serve file server")
		}
	}()

	return s, nil
}

func (s *Storage) Upload(ctx context.Context, filename string, _ int64, uploader string, r io.Reader) (string, error) {
	logger := zerolog.Ctx(ctx)

	logger.Debug().Str("filename", filename).Msg("Uploading file")

	content, err := io.ReadAll(r)
	if err != nil {
		return "", errorx.Wrap(err, "read object")
	}

	logger.Debug().Str("filename", filename).
		Int("size", len(content)).Msg("Uploaded file")

	objID := uuid.New().String()

	s.objs[objID] = object{
		data:     content,
		filename: filename,
		uploader: uploader,
	}

	return objID, nil
}

func (s *Storage) Stat(ctx context.Context, path string) (*storage.Metadata, error) {
	o, ok := s.objs[path]
	if !ok {
		return nil, os.ErrNotExist
	}

	zerolog.Ctx(ctx).Debug().Str("path", path).Msg("Statted file")

	return &storage.Metadata{
		Filename: o.filename,
		Size:     int64(len(o.data)),
		Uploader: o.uploader,
	}, nil
}

func (s *Storage) Download(ctx context.Context, path string) (*storage.Object, error) {
	o, ok := s.objs[path]
	if !ok {
		return nil, os.ErrNotExist
	}

	zerolog.Ctx(ctx).Debug().Str("path", path).Msg("Downloaded file")

	return &storage.Object{
		ReadCloser: io.NopCloser(bytes.NewBuffer(o.data)),
		Metadata: &storage.Metadata{
			Filename: o.filename,
			Size:     int64(len(o.data)),
			Uploader: o.uploader,
		},
	}, nil
}

func (s *Storage) GetLink(_ context.Context, path string) (string, error) {
	return fmt.Sprintf("http://127.0.0.1:%d/%s", s.port, path), nil
}
