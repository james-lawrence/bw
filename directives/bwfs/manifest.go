package bwfs

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// Archive provides details about how to handle a particular file or archive.
// This includes its URI, its Path (destination), the permissions of the root,
// The owner and the group for all files.
type Archive struct {
	URI   string
	Path  string
	Mode  uint32
	Owner string
	Group string
}

func (t Archive) String() string {
	parts := []string{
		t.URI,
		t.Path,
		prettyMode(t.Mode),
		t.Owner,
		t.Group,
	}

	return strings.Join(parts, " ")
}

// Manifest is a line by line specification of archives to download and where to put them.
type Manifest struct {
	Blobs map[string]Archive
}

// ParseManifest parses the given reader and generates a manifest.
func ParseManifest(defaults Archive, r io.Reader) (m []Archive, err error) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		a := defaults
		if err = parse(lex(scanner.Text()), &a); err != nil {
			return m, err
		}

		m = append(m, a)
	}

	return m, scanner.Err()
}

func parseArchiveLine(in string) (a Archive, err error) {
	return Archive{}, nil
}

func prettyMode(m uint32) string {
	return fmt.Sprintf("%04o", m)
}
