#!/bin/sh
set -e

cd /app
OUTDIR="dist/$(apk --print-arch)"

if [ ! -d "$OUTDIR" ] || [ -z "$(ls "$OUTDIR"/*.apk 2>/dev/null)" ]; then
	echo "no packages in $OUTDIR - run 'make apk' first" >&2
	exit 1
fi

mkdir -p /home/builder/.abuild
cp routier.rsa routier.rsa.pub /home/builder/.abuild/
chmod 600 /home/builder/.abuild/routier.rsa
chmod 644 /home/builder/.abuild/routier.rsa.pub
echo "PACKAGER_PRIVKEY=\"/home/builder/.abuild/routier.rsa\"" > /home/builder/.abuild/abuild.conf
chown -R builder:builder /home/builder/.abuild "$OUTDIR"

if command -v openssl >/dev/null 2>&1; then
	openssl rsa -in /home/builder/.abuild/routier.rsa -pubout 2>/dev/null > /tmp/signer.pub || true
	if [ -s /tmp/signer.pub ] && ! cmp -s /tmp/signer.pub /home/builder/.abuild/routier.rsa.pub; then
		echo "ERROR: routier.rsa.pub is not the pair of routier.rsa - clients would get BAD signature" >&2
		exit 1
	fi
fi

cp /home/builder/.abuild/routier.rsa.pub /etc/apk/keys/
cp /home/builder/.abuild/routier.rsa.pub "$OUTDIR/" || true

su builder -c "cd /app/$OUTDIR && apk index --rewrite-arch \$(apk --print-arch) -o APKINDEX.tar.gz *.apk" 2>&1 \
	| grep -v 'WARNING: No provider\|WARNING: Total of.*unsatisfiable\|Your repository may be broken' || true
su builder -c "cd /app/$OUTDIR && abuild-sign APKINDEX.tar.gz"

chown -R 0:0 /app/dist

echo "apk index ready: $OUTDIR/APKINDEX.tar.gz"
echo "signing key routier.rsa.pub sha256: $(sha256sum "$OUTDIR/routier.rsa.pub" | awk '{print $1}')"
echo "the client's /etc/apk/keys/routier.rsa.pub must have the same sha256"
