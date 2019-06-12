COMMIT ?= HEAD
PACKAGE ?= github.com/james-lawrence/bw/cmd
VERSION = $(shell git describe --always --tags --long $(COMMIT))
RELEASE ?= 0.1.$(shell git show -s --format=%ct-%h $(COMMIT))
BW_VERSION ?= 0.1.$(shell git show -s --format=%ct $(COMMIT))
LDFLAGS ?= "-X github.com/james-lawrence/bw/cmd.Version=$(RELEASE)"
release-dev-setup:
	sudo docker build -t debian-build -f .dist/deb/Dockerfile .

generate:
	go generate ./...

install:
	go install -ldflags=$(LDFLAGS) $(PACKAGE)/...

build: generate install

install-debug:
	go install -tags="debug.enabled" -ldflags=$(LDFLAGS) $(PACKAGE)/...

test:
	ginkgo -r -p -keepGoing .

release-push: release
	git tag --force $(RELEASE)
	# dput -f -c .dist/deb/dput.config bw .dist/build/bearded-wookie_$(BW_VERSION)_source.changes
	# hub release create -a .dist/build/bearded-wookie-linux-amd64-$(RELEASE).tar.gz $(RELEASE) -m "linux-amd64-$(RELEASE)"
	echo "released version: $(RELEASE) completed successfully"

release-check:
ifeq ($(origin ALLOW_DIRTY), undefined)
	git diff --exit-code --quiet || { echo repository has uncommitted files. set ALLOW_DIRTY to ignore this check; exit 1; }
endif

release: generate release-check
	git log $(shell git describe --tags --abbrev=0)..HEAD > .dist/RELEASE-NOTES.md
	git add .dist/RELEASE-NOTES.md; git commit -m "release $(RELEASE)";

	GOBIN=$(CURDIR)/.dist/build/bearded-wookie-linux-amd64-$(RELEASE) GOARCH=amd64 GOOS=linux go install -ldflags=$(LDFLAGS) $(PACKAGE)/bw

	git archive --format=tar -o $(CURDIR)/.dist/build/bearded-wookie-source-$(RELEASE).tar HEAD
	tar -f $(CURDIR)/.dist/build/bearded-wookie-source-$(RELEASE).tar --delete '.dist' --delete '.test'
	gzip -f $(CURDIR)/.dist/build/bearded-wookie-source-$(RELEASE).tar

	tar -C .dist --xform 's:^\./::' -czvf .dist/build/bearded-wookie-linux-amd64-$(RELEASE).tar.gz \
		RELEASE-NOTES.md \
		systemd \
		-C build/bearded-wookie-linux-amd64-$(RELEASE) .
	sudo docker run \
		-e BUILD_VERSION=$(RELEASE) \
		-e BW_VERSION=$(BW_VERSION) \
		-e BW_LDFLAGS=$(LDFLAGS) \
		-e DEBEMAIL="$(shell git config user.email)" \
		-e DEBFULLNAME="$(shell git config user.name)" \
		-v $(CURDIR):/opt/bw \
		-v $(HOME)/.gnupg:/root/.gnupg \
		-it debian-build:latest

	dput -f -c .dist/deb/dput.config bw .dist/build/bearded-wookie_$(BW_VERSION)_source.changes

release-clean:
	rm -rf .dist/build
