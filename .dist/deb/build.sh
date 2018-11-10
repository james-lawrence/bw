#!/bin/bash
set -e

export CHANGELOG_DATE=$(date +"%a, %d %b %Y %T %z")
export DEBUILD_DPKG_BUILDPACKAGE_OPTS="-k'${DEBFULLNAME} <${DEBEMAIL}>' -sa"
ARCHIVE=../build/bearded-wookie-linux-amd64-${BUILD_VERSION}.tar.gz
DEBDIR=debian

mkdir -p ${DEBDIR}

cat .templates/control.tmpl | envsubst > ${DEBDIR}/control
cat .templates/changelog.tmpl | envsubst > ${DEBDIR}/changelog
cat .templates/install.tmpl > ${DEBDIR}/install
cat .templates/rules.tmpl > ${DEBDIR}/rules
echo "9" > ${DEBDIR}/compat
tar -C /usr/bin -xf ${ARCHIVE} bw

debuild -S
mv ../bearded-wookie* ../build
