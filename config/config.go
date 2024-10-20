package config

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"reflect"
	"strings"

	"github.com/hashicorp/vault-client-go"
	"github.com/joomcode/errorx"
	"github.com/mitchellh/mapstructure"
	"gopkg.in/yaml.v3"
)

const (
	ConfigPathEnv = "CONFIG_PATH"
)

var (
	ErrNotPointer        = errors.New("dest has to be a pointer")
	ErrUnsupportedFormat = errors.New("unsupported file format")
)

func LoadConfig(ctx context.Context, filename string, dest any) error {
	return LoadConfigVault(ctx, nil, filename, dest)
}

func LoadConfigVault(ctx context.Context, vc *vault.Client, filename string, dest any) error {
	if reflect.TypeOf(dest).Kind() != reflect.Pointer {
		return ErrNotPointer
	}

	unint, err := loadConfigFS(filename)
	if err != nil {
		return errorx.Decorate(err, "load uninterpolated config")
	}

	interpolated, err := interpolate(ctx, vc, unint)
	if err != nil {
		return errorx.Decorate(err, "interpolate config")
	}

	err = mapstructure.Decode(interpolated, dest)
	if err != nil {
		return errorx.Decorate(err, "decode interpolated map")
	}

	return nil
}

//nolint:ireturn // we don't know the structure yet.
func loadConfigFS(fspath string) (any, error) {
	if envPath := os.Getenv(ConfigPathEnv); envPath != "" {
		fspath = envPath
	}

	f, err := os.Open(fspath)
	if err != nil {
		return nil, errorx.Decorate(err, "open file")
	}

	defer func() {
		err = f.Close()
		if err != nil {
			slog.Info("Error closing file", "error", err)
		}
	}()

	var tempdest map[string]any

	fnameSplit := strings.Split(fspath, ".")
	ext := fnameSplit[len(fnameSplit)-1]

	switch ext {
	case "yaml", "yml":
		dc := yaml.NewDecoder(f)
		err = dc.Decode(&tempdest)
	case "json":
		dc := json.NewDecoder(f)
		err = dc.Decode(&tempdest)
	default:
		return nil, ErrUnsupportedFormat
	}

	if err != nil {
		return nil, errorx.Decorate(err, "decode file")
	}

	return tempdest, nil
}
