package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/sovamorco/errorx"
)

type MissingEnvError struct {
	EnvVarName string
}

func (e MissingEnvError) Error() string {
	return fmt.Sprintf("required environment variable \"%s\" is not defined", e.EnvVarName)
}

//nolint:ireturn // we return the same structure we get in, but this cannot be easily updated to use generics.
func interpolate(ctx context.Context, vi any) (any, error) {
	var err error

	switch v := vi.(type) {
	case map[string]any:
		for k, vv := range v {
			v[k], err = interpolate(ctx, vv)
			if err != nil {
				return nil, errorx.Decorate(err, "interpolate map value")
			}
		}
	case []any:
		for i, vv := range v {
			v[i], err = interpolate(ctx, vv)
			if err != nil {
				return nil, errorx.Decorate(err, "interpolate slice value")
			}
		}
	case string:
		return interpolateString(ctx, v)
	}

	return vi, nil
}

//nolint:ireturn // return type depends on interpolator.
func interpolateString(ctx context.Context, v string) (any, error) {
	switch {
	case strings.HasPrefix(v, "ENV->"):
		return EnvInterpolator(strings.TrimPrefix(v, "ENV->"))
	case strings.HasPrefix(v, "OENV->"):
		return OEnvInterpolator(strings.TrimPrefix(v, "OENV->")), nil
	case strings.HasPrefix(v, "FS->"):
		res, err := loadConfigFS(strings.TrimPrefix(v, "FS->"))
		if err != nil {
			return nil, errorx.Decorate(err, "interpolate fs value %s", v)
		}

		return interpolate(ctx, res)
	case strings.HasPrefix(v, "OFS->"):
		res, err := loadConfigFS(strings.TrimPrefix(v, "OFS->"))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				//nolint:nilnil // the return value is literally nil in this case.
				return nil, nil
			}

			return nil, errorx.Decorate(err, "interpolate fs value: %s", v)
		}

		return interpolate(ctx, res)
	}

	return v, nil
}

func EnvInterpolator(inp string) (string, error) {
	val := os.Getenv(inp)
	if val == "" {
		return "", MissingEnvError{EnvVarName: inp}
	}

	return val, nil
}

func OEnvInterpolator(inp string) string {
	return os.Getenv(inp)
}
