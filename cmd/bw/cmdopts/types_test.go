package cmdopts_test

import (
	"regexp"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/james-lawrence/bw/cmd/bw/cmdopts"
	"github.com/stretchr/testify/require"
)

type regexpcli struct {
	Names []*regexp.Regexp `name:"name"`
}

func parseRegexp(t *testing.T, args ...string) (*regexpcli, error) {
	t.Helper()

	c := new(regexpcli)
	k, err := kong.New(c, kong.Exit(func(int) {}), cmdopts.Mappers())
	require.NoError(t, err)

	_, err = k.Parse(args)
	return c, err
}

func TestParseRegexp(t *testing.T) {
	t.Run("repeated flags", func(t *testing.T) {
		c, err := parseRegexp(t, "--name=^web-", "--name=db$")
		require.NoError(t, err)
		require.Len(t, c.Names, 2)
		require.True(t, c.Names[0].MatchString("web-1"))
		require.True(t, c.Names[1].MatchString("primary-db"))
		require.False(t, c.Names[0].MatchString("db"))
	})

	t.Run("invalid expression is an error", func(t *testing.T) {
		_, err := parseRegexp(t, "--name=(")
		require.Error(t, err)
	})
}
