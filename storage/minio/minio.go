package minio

import (
	"context"
	"encoding/base64"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sovamorco/errorx"
	"github.com/sovamorco/gommon/storage"
)

const (
	defaultPublicLinkDuration = 25 * time.Hour
)

//nolint:gochecknoinits // driver pattern.
func init() {
	storage.Register("minio", newMinio)
}

type Storage struct {
	client             *minio.Client
	bucketName         string
	publicLinkDuration time.Duration
}

//nolint:ireturn // required by storage.Register.
func newMinio(_ context.Context, connst string) (storage.Storage, error) {
	u, err := url.Parse(connst)
	if err != nil {
		return nil, errorx.Wrap(err, "parse connection url")
	}

	pw, _ := u.User.Password()

	//nolint:exhaustruct
	client, err := minio.New(u.Host, &minio.Options{
		Creds:  credentials.NewStaticV4(u.User.Username(), pw, ""),
		Secure: u.Scheme == "https",
	})

	pld := defaultPublicLinkDuration

	if qplds := u.Query().Get("public_link_duration"); qplds != "" {
		qpld, err := time.ParseDuration(qplds)
		if err != nil {
			return nil, errorx.Wrap(err, "parse public link duration")
		}

		pld = qpld
	}

	return &Storage{
		client:             client,
		bucketName:         strings.TrimPrefix(u.Path, "/"),
		publicLinkDuration: pld,
	}, errorx.Wrap(err, "create minio client")
}

func (s *Storage) Upload(ctx context.Context, filename string,
	size int64, uploader string, r io.Reader,
) (string, error) {
	fileID := uuid.New().String()

	//nolint:exhaustruct
	_, err := s.client.PutObject(ctx, s.bucketName, fileID, r, size, minio.PutObjectOptions{
		UserMetadata: map[string]string{
			"Filenamebase64": base64.StdEncoding.EncodeToString([]byte(filename)),
			"Uploader":       uploader,
		},
	})

	return fileID, errorx.Wrap(err, "put object")
}

func (s *Storage) Stat(ctx context.Context, path string) (*storage.Metadata, error) {
	//nolint:exhaustruct
	info, err := s.client.StatObject(ctx, s.bucketName, path, minio.StatObjectOptions{})
	if err != nil {
		return nil, errorx.Wrap(err, "stat object")
	}

	filename, ok := info.UserMetadata["Filename"]
	if !ok {
		filenameb64, ok := info.UserMetadata["Filenamebase64"]
		if !ok {
			return nil, errorx.ExternalError.New("missing filename metadata")
		}

		decoded, err := base64.StdEncoding.DecodeString(filenameb64)
		if err != nil {
			return nil, errorx.Wrap(err, "decode filename")
		}

		filename = string(decoded)
	}

	uploader, ok := info.UserMetadata["Uploader"]
	if !ok {
		return nil, errorx.ExternalError.New("missing uploader metadata")
	}

	return &storage.Metadata{
		Filename: filename,
		Uploader: uploader,
		Size:     info.Size,
	}, nil
}

func (s *Storage) Download(ctx context.Context, path string) (*storage.Object, error) {
	md, err := s.Stat(ctx, path)
	if err != nil {
		return nil, errorx.Wrap(err, "stat object")
	}

	//nolint:exhaustruct
	r, err := s.client.GetObject(ctx, s.bucketName, path, minio.GetObjectOptions{})
	if err != nil {
		return nil, errorx.Wrap(err, "get object")
	}

	return &storage.Object{
		ReadCloser: r,
		Metadata:   md,
	}, nil
}

func (s *Storage) GetLink(ctx context.Context, path string) (string, error) {
	u, err := s.client.PresignedGetObject(ctx, s.bucketName, path, s.publicLinkDuration, nil)
	if err != nil {
		return "", errorx.Wrap(err, "get presigned object url")
	}

	return u.String(), nil
}
