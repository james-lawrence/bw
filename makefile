COMMIT ?= HEAD
PACKAGE ?= github.com/james-lawrence/bw/cmd
VERSION = $(shell git describe --always --tags --long $(COMMIT))
RELEASE ?= 0.1.$(shell git show -s --format=%ct-%h $(COMMIT))
BW_VERSION ?= 0.1.$(shell git show -s --format=%ct $(COMMIT))
LDFLAGS ?= ""

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

release: release-check
	rm -rf .dist/build && mkdir -p .dist/build
	mkdir -p .dist/cache

	git log $(shell git describe --tags --abbrev=0)..HEAD > .dist/RELEASE-NOTES.md

	# build amd64 tar bundle
	rsync --exclude '.gitignore' --recursive $(CURDIR)/.dist/linux/ $(CURDIR)/.dist/build/bearded-wookie-linux-amd64-$(RELEASE)/
	GOBIN=$(CURDIR)/.dist/build/bearded-wookie-linux-amd64-$(RELEASE)/usr/local/bin GOARCH=amd64 GOOS=linux go install -ldflags=$(LDFLAGS) $(PACKAGE)/bw
	tar -C .dist --xform 's:^\./::' -czf .dist/build/bearded-wookie-linux-amd64-$(RELEASE).tar.gz \
		RELEASE-NOTES.md \
		-C build/bearded-wookie-linux-amd64-$(RELEASE) .
	# tar -f .dist/build/bearded-wookie-linux-amd64-$(RELEASE).tar.gz  --delete '.test'

	# git archive --format=tar -o $(CURDIR)/.dist/build/bearded-wookie-source-$(RELEASE).tar HEAD
	rm -rf $(CURDIR)/.dist/build/bearded-wookie-source-$(RELEASE)
	# git bundle create $(CURDIR)/.dist/build/git.bundle HEAD
	# git clone $(CURDIR)/.dist/build/git.bundle
	git clone --depth 1 file://$(CURDIR) $(CURDIR)/.dist/build/bearded-wookie-source-$(RELEASE)
	tar -C .dist --xform 's:^\./::' \
		--exclude=".test" \
		--exclude=".examples" \
		-czf .dist/build/bearded-wookie-source-$(RELEASE).tar.gz \
		RELEASE-NOTES.md \
		-C build/bearded-wookie-source-$(RELEASE) .

	docker run \
		--user $(shell id -u):$(shell id -g) \
		-e BUILD_VERSION=$(RELEASE) \
		-e BW_VERSION=$(BW_VERSION) \
		-e BW_LDFLAGS=$(LDFLAGS) \
		-e CACHE_DIR="/opt/bw/.dist/cache" \
		-e HOME="/opt/bw/.dist/cache" \
		-e DEBEMAIL="$(shell git config user.email)" \
		-e DEBFULLNAME="$(shell git config user.name)" \
		-v $(CURDIR):/opt/bw \
		-v $(HOME)/.gnupg:/opt/bw/.dist/cache/.gnupg \
		-it debian-build:latest

	git add .dist/RELEASE-NOTES.md && git commit -m "release $(RELEASE)";

release-clean:
	rm -rf .dist/build
