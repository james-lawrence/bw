package bwfs_test

import (
	"io"
	"log"
	"os"
	"path/filepath"

	. "github.com/james-lawrence/bw/directives/bwfs"

	. "github.com/james-lawrence/bw/internal/gomegax"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Bwfs", func() {
	var (
		tmpdir string
		execer Executer
	)

	BeforeEach(func() {
		var (
			err error
		)

		if tmpdir, err = os.MkdirTemp(".", "test"); err != nil {
			Expect(err).ToNot(HaveOccurred())
		}

		execer = New(log.New(io.Discard, "TEST ", log.LstdFlags), ".fixtures")
	})

	AfterEach(func() {
		os.RemoveAll(tmpdir)
	})

	It("should properly copy a directory", func() {
		archive := Archive{
			Owner: os.Getenv("USER"),
			Group: os.Getenv("USER"),
			Mode:  0766,
			Path:  filepath.Join(tmpdir, "sample-directory"),
			URI:   "sample-directory",
		}
		Expect(execer.Execute(archive)).ToNot(HaveOccurred())
		Expect(archive.Path).To(BeADirectory())
		Expect(archive.Path).To(HaveFilePermissions(os.FileMode(archive.Mode)))
		Expect(filepath.Join(archive.Path, "dir1")).To(BeADirectory())
		Expect(filepath.Join(archive.Path, "dir2")).To(BeADirectory())
		Expect(filepath.Join(archive.Path, "sample-file.txt")).To(BeAnExistingFile())
		Expect(filepath.Join(archive.Path, "dir1", "sample-file.txt")).To(BeAnExistingFile())
		Expect(filepath.Join(archive.Path, "dir2", "sample-file.txt")).To(BeAnExistingFile())
		Expect(archive.Path).To(HaveFilePermissions(os.FileMode(archive.Mode)))
	})

	It("should properly copy a file", func() {
		archive := Archive{
			Owner: os.Getenv("USER"),
			Group: os.Getenv("USER"),
			Mode:  0666,
			Path:  filepath.Join(tmpdir, "sample-file.txt"),
			URI:   "sample-file.txt",
		}
		Expect(execer.Execute(archive)).ToNot(HaveOccurred())
		Expect(archive.Path).To(BeAnExistingFile())
		Expect(archive.Path).To(HaveFilePermissions(os.FileMode(archive.Mode)))
	})

	It("should properly copy a file", func() {
		archive := Archive{
			Owner: os.Getenv("USER"),
			Group: os.Getenv("USER"),
			Mode:  0600,
			Path:  filepath.Join(tmpdir, "sample-file.txt"),
			URI:   "sample-file.txt",
		}
		Expect(execer.Execute(archive)).ToNot(HaveOccurred())
		Expect(archive.Path).To(BeAnExistingFile())
		Expect(archive.Path).To(HaveFilePermissions(os.FileMode(archive.Mode)))
	})
})
