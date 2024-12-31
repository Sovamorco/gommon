package config_test

import (
	"context"
	"testing"
	"time"

	"github.com/sovamorco/gommon/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSuffixes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		inp         any
		expected    any  `exhaustruct:"optional"`
		errExpected bool `exhaustruct:"optional"`
	}{
		{
			name:     "Int from string",
			inp:      "100::atoi",
			expected: 100,
		},
		{
			name:     "Duration from int",
			inp:      "100::atoi::duration",
			expected: time.Duration(100),
		},
		{
			name:     "Duration from string",
			inp:      "1s::duration",
			expected: 1 * time.Second,
		},
		{
			name:        "Duration from invalid value",
			inp:         "test::duration",
			errExpected: true,
		},
		{
			name:        "Int from invalid value",
			inp:         "test::atoi",
			errExpected: true,
		},
		{
			name:        "Chained errors",
			inp:         "test::atoi::duration::atoi::duration",
			errExpected: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			testSuffix(t, c.inp, c.expected, c.errExpected)
		})
	}
}

func testSuffix(t *testing.T, inp, expected any, errExpected bool) {
	t.Helper()

	res, err := config.Interpolate(context.Background(), inp)

	if errExpected {
		require.Error(t, err)
	} else {
		require.NoError(t, err)
		assert.Equal(t, expected, res)
	}
}
