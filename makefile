COMMIT ?= HEAD
PACKAGE ?= github.com/james-lawrence/bw/cmd
VERSION = $(shell git describe --always --tags --long $(COMMIT))
RELEASE ?= 0.1.$(shell git show -s --format=%ct-%h $(COMMIT))
BW_VERSION ?= 0.1.$(shell git show -s --format=%ct $(COMMIT))
LDFLAGS ?= ""

generate:
	go generate ./...

install:
	go install -ldflags=$(LDFLAGS) $(PACKAGE)/...

build: generate install

release-check:
ifeq ($(origin ALLOW_DIRTY), undefined)
	git diff --exit-code --quiet || { echo repository has uncommitted files. set ALLOW_DIRTY to ignore this check; exit 1; }
endif

release: release-check
	git log $(shell git describe --tags --abbrev=0)..HEAD > .dist/RELEASE-NOTES.md

	eg compute local release/debian --invalidate-cache

	git add .dist/RELEASE-NOTES.md && git commit -m "release $(RELEASE)";
