package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/vault-client-go"
	"github.com/joomcode/errorx"
)

var ErrNoVaultClient = errors.New("vault client not supplied for config with vault interpolations")

type MissingEnvError struct {
	EnvVarName string
}

func (e MissingEnvError) Error() string {
	return fmt.Sprintf("required environment variable \"%s\" is not defined", e.EnvVarName)
}

//nolint:ireturn // we return the same structure we get in, but this cannot be easily updated to use generics.
func interpolate(ctx context.Context, vc *vault.Client, vi any) (any, error) {
	var err error

	switch v := vi.(type) {
	case map[string]any:
		for k, vv := range v {
			v[k], err = interpolate(ctx, vc, vv)
			if err != nil {
				return nil, errorx.Decorate(err, "interpolate map value")
			}
		}
	case []any:
		for i, vv := range v {
			v[i], err = interpolate(ctx, vc, vv)
			if err != nil {
				return nil, errorx.Decorate(err, "interpolate slice value")
			}
		}
	case string:
		return interpolateString(ctx, vc, v)
	}

	return vi, nil
}

//nolint:ireturn // return type depends on interpolator.
func interpolateString(ctx context.Context, vc *vault.Client, v string) (any, error) {
	switch {
	case strings.HasPrefix(v, "ENV->"):
		return EnvInterpolator(strings.TrimPrefix(v, "ENV->"))
	case strings.HasPrefix(v, "OENV->"):
		return OEnvInterpolator(strings.TrimPrefix(v, "OENV->")), nil
	case strings.HasPrefix(v, "VAULT->"):
		res, err := VaultInterpolator(ctx, strings.TrimPrefix(v, "VAULT->"), vc)
		if err != nil {
			return nil, errorx.Decorate(err, "interpolate vault value %s", v)
		}

		return interpolate(ctx, vc, res)
	case strings.HasPrefix(v, "FS->"):
		res, err := loadConfigFS(strings.TrimPrefix(v, "FS->"))
		if err != nil {
			return nil, errorx.Decorate(err, "interpolate fs value %s", v)
		}

		return interpolate(ctx, vc, res)
	}

	return v, nil
}

//nolint:ireturn // return type depends on kind of value in vault.
func VaultInterpolator(ctx context.Context, inp string, client *vault.Client) (any, error) {
	if client == nil {
		return nil, ErrNoVaultClient
	}

	var options []vault.RequestOption

	if strings.HasPrefix(inp, "/") {
		split := strings.SplitN(strings.TrimPrefix(inp, "/"), "/", 2) //nolint:mnd
		options = append(options, vault.WithMountPath(split[0]))
		inp = split[1]
	}

	secret, err := client.Secrets.KvV2Read(ctx, inp, options...)
	if err != nil {
		return nil, errorx.Decorate(err, "read secret")
	}

	return secret.Data.Data, nil
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
