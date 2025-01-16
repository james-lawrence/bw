package userx

import (
	"log"
	"os"
	"os/user"
	"path/filepath"

	"github.com/james-lawrence/bw/internal/envx"
)

func Root() user.User {
	return user.User{
		Gid:     "0",
		Uid:     "0",
		HomeDir: "/root",
	}
}

// CurrentUserOrDefault returns the current user or the default configured user.
func CurrentUserOrDefault(d user.User) (result *user.User) {
	var (
		err error
	)

	if result, err = user.Current(); err != nil {
		log.Println("failed to retrieve current user, using default", err)
		tmp := d
		return &tmp
	}

	return result
}

// DefaultCacheDirectory cache directory for storing data.
func DefaultCacheDirectory(rel ...string) string {
	user := CurrentUserOrDefault(Root())
	if user.Uid == Root().Uid {
		return filepath.Join(envx.String(filepath.Join("/", "var", "cache"), "CACHE_DIRECTORY"), filepath.Join(rel...))
	}

	root := filepath.Join(user.HomeDir, ".cache")

	return filepath.Join(envx.String(root, "CACHE_DIRECTORY", "XDG_CACHE_HOME"), filepath.Join(rel...))
}

// DefaultRuntimeDirectory runtime directory for storing data.
func DefaultRuntimeDirectory(rel ...string) string {
	user := CurrentUserOrDefault(Root())

	if user.Uid == Root().Uid {
		return filepath.Join(envx.String(filepath.Join("/", "run"), "RUNTIME_DIRECTORY", "XDG_RUNTIME_DIR"), filepath.Join(rel...))
	}

	return filepath.Join(envx.String(os.TempDir(), "RUNTIME_DIRECTORY", "XDG_RUNTIME_DIR"), filepath.Join(rel...))
}
