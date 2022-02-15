package autocomplete

import (
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/james-lawrence/bw"
	"github.com/posener/complete"
)

func Deployspaces(args complete.Args) (results []string) {
	root := bw.LocateDeployspace(bw.DefaultDeployspaceConfigDir)
	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if root == path {
			return nil
		}

		if !d.IsDir() {
			return nil
		}

		name := d.Name()

		if strings.HasPrefix(name, args.Last) {
			results = append(results, name)
		}

		return filepath.SkipDir
	})
	return results
}
