#!/bin/bash
set -e

release() {
  VERSION=$2
  DISTRO=$1
  DEBDIR=debian

  echo "DISTRO ${DISTRO} VERSION ${VERSION}"
  mkdir -p ${DEBDIR}
  mkdir -p ${DEBDIR}/source
  mkdir -p src/github.com/james-lawrence/bw

  cat .templates/control.tmpl | envsubst > ${DEBDIR}/control
  cat .templates/changelog.tmpl | env DISTRO=${DISTRO} VERSION=${VERSION} envsubst > ${DEBDIR}/changelog
  cat .templates/install.tmpl > ${DEBDIR}/install
  cat .templates/rules.tmpl > ${DEBDIR}/rules

  echo "9" > ${DEBDIR}/compat
  echo "3.0 (native)" > ${DEBDIR}/source/format

  debuild -S

  mv ../bearded-wookie* ../build

  echo "UPLOAD INITIATED ${DISTRO} - ${VERSION}"
  dput -f -c dput.config bw ../build/bearded-wookie_${VERSION}_source.changes
  echo "UPLOAD COMPLETED ${DISTRO} - ${VERSION}"
}

export CHANGELOG_DATE=$(date +"%a, %d %b %Y %T %z")
export DEBUILD_DPKG_BUILDPACKAGE_OPTS="-k'${DEBFULLNAME} <${DEBEMAIL}>' -sa"
ARCHIVE=../build/bearded-wookie-source-${BUILD_VERSION}.tar.gz

cp ${ARCHIVE} ../bearded-wookie_${VERSION}.orig.tar.gz
tar -xf ../bearded-wookie_${VERSION}.orig.tar.gz -C src/github.com/james-lawrence/bw

i=-1
for distro in "$@"
do
  i=$(( i + 1 ))
  # append the index here to ensure unique versions per distro.
  release "$distro" "${BW_VERSION}${i}"
done
