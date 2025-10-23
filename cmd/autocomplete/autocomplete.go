package autocomplete

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/posener/complete"
)

func Deployspaces(args complete.Args) (results []string) {
	root := bw.LocateDeployspace(bw.DefaultDeployspaceConfigDir)
	errorsx.Fatal(filepath.Walk(root, func(path string, d os.FileInfo, err error) error {
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
	}))
	return results
}
