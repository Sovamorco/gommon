package config

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/hashicorp/vault-client-go"
	"github.com/joomcode/errorx"
	"github.com/mitchellh/mapstructure"
	"gopkg.in/yaml.v3"
	"log/slog"
	"os"
	"reflect"
	"strings"
)

var (
	ErrNotPointer        = errors.New("dest has to be a pointer")
	ErrNoLoader          = errors.New("no suitable loader")
	ErrUnsupportedFormat = errors.New("unsupported file format")
)

type loader func(context.Context, string) (any, error)

func getLoaders() map[string]loader {
	return map[string]loader{
		"CONFIG_PATH": loadConfigFS,
	}
}

func LoadConfig(ctx context.Context, dest any) error {
	return LoadConfigVault(ctx, nil, dest)
}

func LoadConfigVault(ctx context.Context, vc *vault.Client, dest any) error {
	if reflect.TypeOf(dest).Kind() != reflect.Pointer {
		return ErrNotPointer
	}

	loaders := getLoaders()
	for k, loader := range loaders {
		ev := os.Getenv(k)
		if ev != "" {
			unint, err := loader(ctx, ev)
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
	}

	return ErrNoLoader
}

func loadConfigFS(_ context.Context, fspath string) (any, error) {
	f, err := os.Open(fspath)
	if err != nil {
		return nil, errorx.Decorate(err, "open file")
	}

	defer func() {
		err = f.Close()
		if err != nil {
			slog.Info("Error closing file: ", err)
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
