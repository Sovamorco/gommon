package config

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/sovamorco/errorx"
)

type MissingEnvError struct {
	EnvVarName string
}

func (e MissingEnvError) Error() string {
	return fmt.Sprintf("required environment variable \"%s\" is not defined", e.EnvVarName)
}

type SuffixInterpolator func(v any) (any, error)

// and we want to declare a regex at the start so we fail quickly if it does not render.
//
//nolint:gochecknoglobals // need to declare it for the sake of declaring a regex right away
var SuffixInterpolators = map[string]SuffixInterpolator{
	"atoi":     AtoiInterpolator,
	"duration": DurationInterpolator,
}

var SuffixRegex = regexp.MustCompile(fmt.Sprintf(`^(.+)::(%s)$`,
	strings.Join(slices.Collect(maps.Keys(SuffixInterpolators)), "|")))

//nolint:ireturn // we return the same structure we get in, but this cannot be easily updated to use generics.
func Interpolate(ctx context.Context, vi any) (any, error) {
	var err error

	switch v := vi.(type) {
	case map[string]any:
		for k, vv := range v {
			v[k], err = Interpolate(ctx, vv)
			if err != nil {
				return nil, errorx.Decorate(err, "interpolate map value")
			}
		}
	case []any:
		for i, vv := range v {
			v[i], err = Interpolate(ctx, vv)
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
	v, suffixes := getSuffixes(v)

	var res any = v

	var err error

	switch {
	case strings.HasPrefix(v, "ENV->"):
		res, err = EnvInterpolator(strings.TrimPrefix(v, "ENV->"))
	case strings.HasPrefix(v, "OENV->"):
		res, err = OEnvInterpolator(strings.TrimPrefix(v, "OENV->")), nil
	case strings.HasPrefix(v, "FS->"):
		res, err = loadConfigFS(strings.TrimPrefix(v, "FS->"))
	case strings.HasPrefix(v, "OFS->"):
		res, err = loadConfigFS(strings.TrimPrefix(v, "OFS->"))
		if errors.Is(err, os.ErrNotExist) {
			err = nil
		}
	default:
		// if there are no prefixes and no suffixes - return the value.
		if len(suffixes) == 0 {
			return res, nil
		}
	}

	if err != nil {
		return nil, errorx.Decorate(err, "interpolate prefix")
	}

	return interpolateSuffixes(ctx, res, suffixes)
}

//nolint:ireturn // value depends on suffixes.
func interpolateSuffixes(ctx context.Context, v any, suffixes []string) (any, error) {
	var err error

	for _, suffix := range suffixes {
		si, ok := SuffixInterpolators[suffix]
		if !ok {
			return nil, errorx.IllegalState.New("got non-existent suffix interpolator %s", suffix)
		}

		v, err = si(v)
		if err != nil {
			return nil, errorx.Decorate(err, "interpolate suffix %s", suffix)
		}
	}

	return Interpolate(ctx, v)
}

func getSuffixes(v string) (string, []string) {
	res := make([]string, 0)

	for {
		submatch := SuffixRegex.FindStringSubmatch(v)
		if submatch == nil {
			break
		}

		v = submatch[1]
		// append in reverse order because the suffixes are stripped from right to left,
		// and are supposed to be executed from left to right.
		res = append([]string{submatch[2]}, res...)
	}

	return v, res
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

//nolint:ireturn // required by SuffixInterpolator type.
func AtoiInterpolator(inp any) (any, error) {
	switch inpt := inp.(type) {
	case int:
		return inpt, nil
	case string:
		res, err := strconv.Atoi(inpt)
		if err != nil {
			return nil, errorx.Decorate(err, "parse int")
		}

		return res, nil
	}

	return nil, errorx.IllegalArgument.New("value for atoi intepolator has to be a string or an int")
}

//nolint:ireturn // required by SuffixInterpolator type.
func DurationInterpolator(inp any) (any, error) {
	switch inpt := inp.(type) {
	case int:
		return time.Duration(inpt), nil
	case string:
		res, err := time.ParseDuration(inpt)
		if err != nil {
			return nil, errorx.Decorate(err, "parse duration")
		}

		return res, nil
	}

	return nil, errorx.IllegalArgument.New("value for duration intepolator has to be a string or an int")
}
