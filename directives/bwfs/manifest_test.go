package bwfs

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestManifest(t *testing.T) {
	default1 := Archive{
		URI:   "s3://33cd:b41b:526c:7c4d:8532:b150:e8cd:c367?dolor=ipsum",
		Path:  "/root/path",
		Mode:  0755,
		Owner: "root",
		Group: "root",
	}

	for _, c := range []struct {
		name     string
		src      io.Reader
		d        Archive
		expected []Archive
	}{
		{
			"example 1 - parse one per line",
			strings.NewReader(default1.String() + "\n" + default1.String() + "\n"),
			default1,
			[]Archive{default1, default1},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			results, err := ParseManifest(c.d, c.src)
			require.NoError(t, err)
			require.Equal(t, c.expected, results)
		})
	}
}
