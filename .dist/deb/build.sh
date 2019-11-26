#!/bin/bash
set -e

export CHANGELOG_DATE=$(date +"%a, %d %b %Y %T %z")
export DEBUILD_DPKG_BUILDPACKAGE_OPTS="-k'${DEBFULLNAME} <${DEBEMAIL}>' -sa"

ARCHIVE=../build/bearded-wookie-source-${BUILD_VERSION}.tar.gz
DEBDIR=debian
DISTRO=$1
echo "ARGUMENTS $@"
mkdir -p ${DEBDIR}
mkdir -p ${DEBDIR}/source
mkdir -p src/github.com/james-lawrence/bw

cat .templates/control.tmpl | envsubst > ${DEBDIR}/control
cat .templates/changelog.tmpl | envsubst > ${DEBDIR}/changelog
cat .templates/install.tmpl > ${DEBDIR}/install
cat .templates/rules.tmpl > ${DEBDIR}/rules
echo "9" > ${DEBDIR}/compat
echo "3.0 (native)" > ${DEBDIR}/source/format

cp ${ARCHIVE} ../bearded-wookie_${DISTRO}_${BW_VERSION}.orig.tar.gz
tar -xf ../bearded-wookie_${DISTRO}_${BW_VERSION}.orig.tar.gz -C src/github.com/james-lawrence/bw

debuild -S

mv ../bearded-wookie* ../build
