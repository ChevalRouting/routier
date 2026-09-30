#!/bin/sh
set -e

cd /app

OUTDIR="dist/$(apk --print-arch)"

mkdir -p "$OUTDIR"

mkdir -p /work
chown builder:builder /work

cp APKBUILD scripts/routier.post-install scripts/routier.post-upgrade scripts/routier.pre-deinstall /work/

cp -r . /tmp/routier-${VERSION}
rm -rf /tmp/routier-${VERSION}/.git /tmp/routier-${VERSION}/dist
rm -rf /tmp/routier-${VERSION}/web/node_modules
rm -rf /tmp/routier-${VERSION}/pkg/api/dist
rm -f /tmp/routier-${VERSION}/go.work /tmp/routier-${VERSION}/go.work.sum

EXTRA_DIRS=""
if [ "${LOCAL_ANYK}" = "1" ] && [ -d /anyk ]; then
  cp -r /anyk /tmp/anyk
  rm -rf /tmp/anyk/.git
  [ -f go.work ] && cp go.work /tmp/routier-${VERSION}/
  [ -f go.work.sum ] && cp go.work.sum /tmp/routier-${VERSION}/
  EXTRA_DIRS="anyk"
fi

tar czf /work/routier-${VERSION}.tar.gz -C /tmp routier-${VERSION} ${EXTRA_DIRS}
rm -rf /tmp/routier-${VERSION} /tmp/anyk

chown -Rv builder:builder /work

mkdir -p /home/builder/.abuild
cp -v routier.rsa /home/builder/.abuild/
cp -v routier.rsa.pub /home/builder/.abuild/
chmod -v 600 /home/builder/.abuild/routier.rsa
chmod -v 644 /home/builder/.abuild/routier.rsa.pub
echo "PACKAGER_PRIVKEY=\"/home/builder/.abuild/routier.rsa\"" > /home/builder/.abuild/abuild.conf

chown -R builder:builder /home/builder/.abuild

if command -v openssl >/dev/null 2>&1; then
  openssl rsa -in /home/builder/.abuild/routier.rsa -pubout 2>/dev/null > /tmp/signer.pub || true
  if [ -s /tmp/signer.pub ] && ! cmp -s /tmp/signer.pub /home/builder/.abuild/routier.rsa.pub; then
    echo "ERROR: routier.rsa.pub is not the pair of routier.rsa - packages would get UNTRUSTED signature" >&2
    exit 1
  fi
fi

cp /home/builder/.abuild/*.pub /etc/apk/keys/

mkdir -p /app/dist/cache/npm
chmod -R a+rwX /app/dist/cache

rm -rf /home/builder/packages
su builder -c "cd /work && export npm_config_cache=/app/dist/cache/npm && abuild checksum && abuild -F"

find /home/builder/packages -name '*.apk' -exec cp {} "$OUTDIR/" \;
cp /home/builder/.abuild/*.pub "$OUTDIR/" || true

chown -R builder:builder "$OUTDIR"
su builder -c "cd /app/$OUTDIR && apk index --rewrite-arch \$(apk --print-arch) -o APKINDEX.tar.gz *.apk" 2>&1 \
    | awk '
        /^WARNING: No provider/ { skip = 1; next }
        skip && /^[[:space:]]/  { next }
        { skip = 0 }
        /^WARNING: Total of .*unsatisfiable/ { next }
        /Your repository may be broken/ { next }
        { print }
      ' || true
su builder -c "cd /app/$OUTDIR && abuild-sign APKINDEX.tar.gz"

chown -R 0:0 "/app/dist"
