package config

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"reflect"
	"strings"

	"github.com/go-viper/mapstructure/v2"

	"github.com/sovamorco/errorx"
	"gopkg.in/yaml.v3"
)

const (
	ConfigPathEnv = "CONFIG_PATH"
)

var ErrNotPointer = errors.New("dest has to be a pointer")

func LoadConfig(ctx context.Context, filename string, dest any) error {
	if reflect.TypeOf(dest).Kind() != reflect.Pointer {
		return ErrNotPointer
	}

	unint, err := loadConfigFS(ctx, filename)
	if err != nil {
		return errorx.Wrap(err, "load uninterpolated config")
	}

	interpolated, err := Interpolate(ctx, unint)
	if err != nil {
		return errorx.Wrap(err, "interpolate config")
	}

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: decodeBase64Strings,
		Result:     dest,
	})
	if err != nil {
		return errorx.Wrap(err, "create decoder")
	}

	err = decoder.Decode(interpolated)
	if err != nil {
		return errorx.Wrap(err, "decode interpolated map")
	}

	return nil
}

//nolint:ireturn // we don't know the structure yet.
func loadConfigFS(ctx context.Context, fspath string) (any, error) {
	if envPath := os.Getenv(ConfigPathEnv); envPath != "" {
		fspath = envPath
	}

	f, err := os.Open(fspath)
	if err != nil {
		return nil, errorx.Wrap(err, "open file %s", fspath)
	}

	defer func() {
		err = f.Close()
		if err != nil {
			slog.InfoContext(ctx, "Error closing file", "error", err)
		}
	}()

	var tempdest map[string]any

	var res any

	fnameSplit := strings.Split(fspath, ".")
	ext := fnameSplit[len(fnameSplit)-1]

	switch ext {
	case "yaml", "yml":
		dc := yaml.NewDecoder(f)
		err = dc.Decode(&tempdest)
		res = tempdest
	case "json":
		dc := json.NewDecoder(f)
		err = dc.Decode(&tempdest)
		res = tempdest
	case "", "txt":
		res, err = readAllString(f)
	default:
		return nil, errorx.IllegalArgument.New("unsupported file format \"%s\" for file %s", ext, fspath)
	}

	if err != nil {
		return nil, errorx.Wrap(err, "decode file %s", fspath)
	}

	return res, nil
}

func readAllString(r io.Reader) (string, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return "", errorx.Wrap(err, "read all")
	}

	return string(b), nil
}

func decodeBase64Strings(s reflect.Type, d reflect.Type, data any) (any, error) {
	if s.Kind() != reflect.String || d != reflect.SliceOf(reflect.TypeOf(byte(0))) {
		return data, nil
	}

	res, err := base64.StdEncoding.DecodeString(data.(string))

	return res, errorx.Wrap(err, "decode base64 value")
}
