// Package eggit provides functionality around git and assumes that the git command is available.
package eggit

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	_eg "github.com/egdaemon/eg"
	"github.com/egdaemon/eg/internal/debugx"
	"github.com/egdaemon/eg/internal/errorsx"
	"github.com/egdaemon/eg/internal/fsx"
	"github.com/egdaemon/eg/internal/iox"
	"github.com/egdaemon/eg/internal/slicesx"
	"github.com/egdaemon/eg/internal/stringsx"
	"github.com/egdaemon/eg/runtime/wasi/eg"
	"github.com/egdaemon/eg/runtime/wasi/egenv"
	"github.com/egdaemon/eg/runtime/wasi/egunsafe/ffigit"
	"github.com/egdaemon/eg/runtime/wasi/env"
	"github.com/egdaemon/eg/runtime/wasi/shell"
)

const hashSize = 20

type hash [hashSize]byte

// NewHash return a new Hash from a hexadecimal hash representation
func nhash(s string) hash {
	b, _ := hex.DecodeString(s)

	var h hash
	copy(h[:], b)

	return h
}

func (h hash) IsZero() bool {
	var empty hash
	return h == empty
}

func (h hash) String() string {
	return hex.EncodeToString(h[:])
}

func (h hash) Format(s fmt.State, verb rune) {
	switch verb {
	case 's':
		w := hashSize * 2
		if cw, ok := s.Width(); ok {
			w = cw
		}
		encoded := []rune(hex.EncodeToString(h[:]))
		_ = errorsx.Must(s.Write([]byte(string(encoded[:w]))))
	default:
		_ = errorsx.Must(s.Write([]byte(h.String())))
	}
}

type signature struct {
	Name  string
	Email string
	When  time.Time
}

type commit struct {
	// Hash of the commit object.
	Hash hash
	// Author is the original author of the commit.
	Author signature
	// Committer is the one performing the commit, might be different from
	// Author.
	Committer signature
}

// Replace patterns in a string with their values from information in a git commit.
// %git.hash%              -> the full hex encoded commit id.
// %git.hash.short%        -> the first 7 characters of the hex encoded commit id.
// %git.commit.year%       -> year of the commit.
// %git.commit.month%      -> month of the commit.
// %git.commit.day%        -> day of the commit.
// %git.commit.unix.milli% -> UnixMilli of the commit.
// %eg.git.canonical.uri%  -> the uri of the repository that eg considers to be the canonical uri.
func (c commit) StringReplace(pattern string) string {
	cwhen := c.Committer.When

	s := strings.ReplaceAll(pattern, "%git.hash%", c.Hash.String())
	s = strings.ReplaceAll(s, "%git.hash.short%", fmt.Sprintf("%7s", c.Hash))
	s = strings.ReplaceAll(s, "%git.commit.year%", strconv.Itoa(cwhen.Year()))
	s = strings.ReplaceAll(s, "%git.commit.month%", strconv.Itoa(int(cwhen.Month())))
	s = strings.ReplaceAll(s, "%git.commit.day%", strconv.Itoa(cwhen.Day()))
	s = strings.ReplaceAll(s, "%git.commit.unix.milli%", strconv.Itoa(int(cwhen.UnixMilli())))
	s = strings.ReplaceAll(s, "%eg.git.canonical.uri%", EnvCanonicalURI())

	return s
}

// completely clean any git substitutions from the pattern.
func PatternClean(pattern string) string {
	s := strings.ReplaceAll(pattern, ".%", "%")
	s = strings.ReplaceAll(s, "%git.hash%", "")
	s = strings.ReplaceAll(s, "%git.hash.short%", "")
	s = strings.ReplaceAll(s, "%git.commit.year%", "")
	s = strings.ReplaceAll(s, "%git.commit.month%", "")
	s = strings.ReplaceAll(s, "%git.commit.day%", "")
	s = strings.ReplaceAll(s, "%git.commit.unix.milli%", "")
	s = strings.ReplaceAll(s, "%eg.git.canonical.uri%", "")
	return s
}

// substitute values in the provided pattern using the environment variable commit.
// see commit.StringReplace for more details.
func StringReplace(pattern string) string {
	return EnvCommit().StringReplace(pattern)
}

func EnvCanonicalURI() string {
	return os.Getenv(_eg.EnvComputeVCS)
}

// retrieve the commit metadata from from the environment.
func EnvCommit() *commit {
	return &commit{
		Hash: nhash(os.Getenv(_eg.EnvGitHeadCommit)),
		Committer: signature{
			Name:  os.Getenv(_eg.EnvGitHeadCommitAuthor),
			Email: os.Getenv(_eg.EnvGitHeadCommitEmail),
			When:  errorsx.Zero(time.Parse(time.RFC3339, os.Getenv(_eg.EnvGitHeadCommitTimestamp))),
		},
	}
}

// determine the commit hash for the given name.
func Commitish(ctx context.Context, treeish string) string {
	return ffigit.Commitish(ctx, treeish)
}

// clone a git repository from the given uri, remote and treeish.
func Clone(ctx context.Context, uri, remote, commit string) error {
	if strings.TrimSpace(uri) == "" {
		return fmt.Errorf("unable to clone url not specified: %s", uri)
	}

	if strings.TrimSpace(remote) == "" {
		return fmt.Errorf("unable to clone remote not specified: %s", remote)
	}

	return ffigit.Clone(ctx, uri, remote, commit)
}

// clone the repository specified by the eg environment variables into the working directory.
func AutoClone(ctx context.Context, _ eg.Op) error {
	if err := Clone(ctx, env.String("", "EG_GIT_HEAD_URI"), env.String("origin", "EG_GIT_HEAD_REMOTE"), env.String("main", "EG_GIT_HEAD_REF")); err != nil {
		return err
	}

	return nil

}

type modified struct {
	o       sync.Once
	changed []string
}

func (t *modified) detect(ctx context.Context) error {
	var (
		path = egenv.RuntimeDirectory("eg.git.mod")
	)

	hcommit := egenv.String("", _eg.EnvGitHeadCommit)
	bcommit := egenv.String(hcommit, _eg.EnvGitBaseCommit)
	if strings.TrimSpace(hcommit) == "" {
		log.Println(fmt.Errorf("environment variable %s is empty", _eg.EnvGitHeadCommit))
	} else {
		debugx.Println("git.modified head commit", hcommit)
	}
	if strings.TrimSpace(bcommit) == "" {
		log.Println(fmt.Errorf("environment variable %s is empty", _eg.EnvGitBaseCommit))
	} else {
		debugx.Println("git.modified base commit", bcommit)
	}

	if hcommit == bcommit {
		return nil
	}

	// handle detecting missing commits
	// if we can't find one of the commits
	// then we treat it as if the commits were identical
	// resulting everything being treated as changed.
	err := shell.Run(
		ctx,
		shell.Newf(
			"git show --name-only %s > /dev/null 2>&1", bcommit,
		).Directory(egenv.WorkingDirectory()),
		shell.Newf(
			"git show --name-only %s > /dev/null 2>&1", hcommit,
		).Directory(egenv.WorkingDirectory()),
	)
	if err != nil {
		return nil
	}

	err = shell.Run(
		ctx,
		shell.Newf(
			"git diff --name-only %s..%s | tee %s > /dev/null 2>&1", bcommit, hcommit, path,
		).Directory(egenv.WorkingDirectory()),
	)
	if err != nil {
		return errorsx.Wrap(err, "git diff failed")
	}

	mods, err := os.Open(path)
	// if we didnt generate the changeset then no changes occurred.
	if fsx.ErrIsNotExist(err) != nil {
		return nil
	} else if err != nil {
		return err
	}

	smods := iox.String(mods)

	if stringsx.Blank(smods) {
		return nil
	}

	t.changed = strings.Split(smods, "\n")
	return nil
}

func (t *modified) Changed(paths ...string) func(context.Context) bool {
	return func(ctx context.Context) bool {
		t.o.Do(func() {
			errorsx.Log(t.detect(ctx))
		})

		debugx.Println("changed", len(t.changed), t.changed, "paths", len(paths))
		if len(t.changed) == 0 || len(paths) == 0 {
			return true
		}

		return stringsx.Present(slicesx.FindOrZero(func(s string) bool {
			for _, n := range paths {
				if strings.HasPrefix(s, n) {
					return true
				}
			}

			return false
		}, t.changed...))
	}
}

func NewModified() modified {
	return modified{o: sync.Once{}}
}

// ensures the workspace has not been modified. useful for detecting
// if there have been changes during the run.
func Pristine() eg.OpFn {
	return func(ctx context.Context, _ eg.Op) error {
		log.Println("ensuring git repository has not changed initiated")
		defer log.Println("ensuring git repository has not changed completed")
		return shell.Run(
			ctx,
			shell.New("git diff-index --name-only HEAD"),
			shell.New("git diff-index --quiet HEAD"),
		)
	}
}
