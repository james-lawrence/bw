COMMIT ?= HEAD
PACKAGE ?= bitbucket.org/jatone/bearded-wookie/commands
VERSION = $(shell git describe --always --tags --long $(COMMIT))
RELEASE = $(shell git describe --always --tags --long $(COMMIT) | sed 's/\(.*\)-.*/\1/')
LDFLAGS ?= "-X bitbucket.org/jatone/bearded-wookie/commands.Version=$(VERSION)"

generate:
	go list ./... | grep -v /vendor/ | xargs go generate

install: generate
	go install -ldflags=$(LDFLAGS) $(PACKAGE)/...

release-check:
ifeq ($(origin ALLOW_DIRTY), undefined)
	git diff --exit-code --quiet || { echo repository has uncommitted files. set ALLOW_DIRTY to ignore this check; exit 1; }
endif

release: release-check
	echo "released version: $(RELEASE) completed successfully"
