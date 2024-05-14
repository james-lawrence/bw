#!/bin/bash
set -e

release() {
  DISTRO=$1
  VERSION=$2
  BUILDFLAGS=$3
  DEBDIR=debian

  echo "DISTRO ${DISTRO} VERSION ${VERSION} - ${BUILDFLAGS}"
  mkdir -p ${DEBDIR}
  mkdir -p ${DEBDIR}/source

  cat .templates/control.tmpl | envsubst > ${DEBDIR}/control
  cat .templates/changelog.tmpl | env DISTRO=${DISTRO} VERSION=${VERSION} envsubst > ${DEBDIR}/changelog
  cat .templates/install.tmpl > ${DEBDIR}/install
  cat .templates/rules.tmpl | env BW_LDFLAGS="${BUILDFLAGS}" envsubst > ${DEBDIR}/rules

  echo "10" > ${DEBDIR}/compat
  echo "3.0 (native)" > ${DEBDIR}/source/format

  debuild -S

  mv ../bearded-wookie* ../build

  echo "UPLOAD INITIATED ${DISTRO} - ${VERSION}"
  dput -f -c dput.config bw ../build/bearded-wookie_${VERSION}_source.changes
  echo "UPLOAD COMPLETED ${DISTRO} - ${VERSION}"
}

rm -rf src && mkdir -p src
rm -rf deb

export USER="root"
export GOBIN="${HOME}/go/bin"
export CHANGELOG_DATE=$(date +"%a, %d %b %Y %T %z")
export DEBUILD_DPKG_BUILDPACKAGE_OPTS="-k'${DEBFULLNAME} <${DEBEMAIL}>' -sa"
ARCHIVE=../build/bearded-wookie-source-${BUILD_VERSION}.tar.gz

cp ${ARCHIVE} ../bearded-wookie_${VERSION}.orig.tar.gz

tar -xf ../bearded-wookie_${VERSION}.orig.tar.gz -C src/

pushd src; /usr/lib/go-1.21/bin/go install -buildvcs=false ./cmd/...; popd
~/go/bin/bw version

i=-1
for distro in "$@"
do
  i=$(( i + 1 ))
  # append the index here to ensure unique versions per distro.
  release "$distro" "${BW_VERSION}${i}" "${BW_LDFLAGS}"
done
