COMMIT ?= HEAD
PACKAGE ?= github.com/james-lawrence/bw/commands
VERSION = $(shell git describe --always --tags --long $(COMMIT))
RELEASE ?= 0.1.$(shell git show -s --format=%ct-%h $(COMMIT))
LDFLAGS ?= "-X github.com/james-lawrence/bw/commands.Version=$(RELEASE)"

generate:
	go generate ./...

install: generate
	go install -ldflags=$(LDFLAGS) $(PACKAGE)/...

install-debug:
	go install -tags="debug.enabled" -ldflags=$(LDFLAGS) $(PACKAGE)/...

test:
	ginkgo -r -p -keepGoing .

release-push: release
	git tag --force $(RELEASE)
	hub release create -a .dist/bearded-wookie-linux-amd64-$(RELEASE).tar.gz $(RELEASE) -m "linux-amd64-$(RELEASE)"
	echo "released version: $(RELEASE) completed successfully"

release-check:
ifeq ($(origin ALLOW_DIRTY), undefined)
	git diff --exit-code --quiet || { echo repository has uncommitted files. set ALLOW_DIRTY to ignore this check; exit 1; }
endif

release: generate release-check
	GOBIN=$(CURDIR)/.dist/bearded-wookie-linux-amd64-$(RELEASE) GOARCH=amd64 GOOS=linux go install -ldflags=$(LDFLAGS) $(PACKAGE)/bw
	tar -C .dist/ -czvf .dist/bearded-wookie-linux-amd64-$(RELEASE).tar.gz ../RELEASE-NOTES.md bearded-wookie-linux-amd64-$(RELEASE)
