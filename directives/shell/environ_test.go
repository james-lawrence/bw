package shell_test

import (
	"io"
	"log"
	"os"
	"sort"
	"testing"

	"github.com/james-lawrence/bw/directives/shell"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	log.SetOutput(io.Discard)
	os.Exit(m.Run())
}

func mustReadString(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}

	return string(raw)
}

func fromMap(m map[string]string) string {
	env, err := godotenv.Marshal(m)
	if err != nil {
		panic(err)
	}

	return env
}

func TestEnvironParsing(t *testing.T) {
	for _, c := range []struct {
		name     string
		example  string
		expected []string
	}{
		{"basic environment", "FOO=BAR", []string{"FOO=BAR"}},
		{"quoted environment", "FOO=\"HELLO WORLD\"", []string{"FOO=\"HELLO WORLD\""}},
		{"multiline environment 1", fromMap(map[string]string{"FOO": "HELLO\nWORLD"}), []string{"FOO=\"HELLO\nWORLD\""}},
		{"multiline environment 2", "FOO=\"HELLO\\nWORLD\"", []string{"FOO=\"HELLO\nWORLD\""}},
		{
			"environmnet file example",
			mustReadString(".fixtures/environ.env"),
			[]string{
				"SIMPLE=VALUE",
				"QUOTED=\"QUOTED VALUE\"",
				"MULTILINE=\"Line1\nLine2\"",
			},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			result := shell.Environ(c.example)
			sort.Strings(result)
			sort.Strings(c.expected)
			require.Equal(t, c.expected, result)
		})
	}
}
