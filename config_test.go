package bw_test

import (
	. "github.com/james-lawrence/bw"
	"github.com/stretchr/testify/require"
	"testing"
)

func environmentSet1(k string) string {
	switch k {
	case "FOO":
		return "BAR"
	case "BIZZ":
		return "BAZZ"
	case "MULTILINE":
		return "line1\nline2\nline3"
	default:
		return ""
	}
}

type xType struct {
	Field1 string
	Field2 string
	Field3 string
}

func TestConfig(t *testing.T) {
	t.Run("ExpandEnvironAndDecode", func(t *testing.T) {
		check := func(t *testing.T, content string, result xType) {
			out := xType{}
			require.NoError(t, ExpandEnvironAndDecode([]byte(content), &out, environmentSet1))
			require.Equal(t, result, out)
		}
		t.Run("example 1", func(t *testing.T) {
			check(t, `field1: "${FOO}"
field2: "${BIZZ}"
field3: "YOINK"
`, xType{
				Field1: "BAR",
				Field2: "BAZZ",
				Field3: "YOINK",
			})
		})
		t.Run("example 2", func(t *testing.T) {
			check(t, `field1: '${MULTILINE}'`, xType{
				Field1: "line1\nline2\nline3",
			})
		})
	})
}
