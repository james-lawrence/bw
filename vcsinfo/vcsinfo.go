package vcsinfo

import (
	"fmt"
	"log"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/internal/stringsx"
)

func Commitish(dir, treeish string) string {
	var (
		err  error
		r    *git.Repository
		hash *plumbing.Hash
	)

	if r, err = git.PlainOpen(dir); err != nil {
		log.Println("unable to detect git repository - commit will be empty", dir, err)
		return ""
	}

	if hash, err = r.ResolveRevision(plumbing.Revision(treeish)); err != nil {
		log.Println("unable to resolve git reference - commit will be empty", dir, treeish, err)
		return ""
	}

	return hash.String()
}

// returns the username and email as a string
func CurrentUserDisplay(dir string) string {
	var (
		err error
		r   *git.Repository
		c   *config.Config
	)

	if r, err = git.PlainOpen(dir); err != nil {
		log.Println("unable to detect git repository - using system default", dir, err)
		return bw.DisplayName()
	}

	if c, err = r.ConfigScoped(config.GlobalScope); err != nil {
		log.Println("unable to load configuration for git repository - using system default", dir, err)
		return bw.DisplayName()
	}

	if stringsx.Empty(c.User.Email) {
		log.Println("git user.email is missing - using system default", dir)
		return stringsx.DefaultIfBlank(c.User.Name, bw.DisplayName())
	}

	return fmt.Sprintf("%s <%s>", stringsx.DefaultIfBlank(c.User.Name, bw.DisplayName()), c.User.Email)
}
