package vcsinfo

import (
	"log"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

func Commitish(dir, treeish string) string {
	var (
		err error
		r   *git.Repository
		ref *plumbing.Reference
	)
	if r, err = git.PlainOpen(dir); err != nil {
		log.Println("unable to detect git repository - commit will be empty", dir, err)
		return ""
	}

	if ref, err = r.Reference(plumbing.ReferenceName(treeish), true); err != nil {
		log.Println("unable to resolve git reference - commit will be empty", dir, err)
		return ""
	}

	return ref.Hash().String()
}