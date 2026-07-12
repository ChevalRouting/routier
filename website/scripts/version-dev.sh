#!/bin/sh
set -e
cd "$(dirname "$0")/.."

rm -rf versioned_docs/version-dev versioned_sidebars/version-dev-sidebars.json

if [ -f versions.json ]; then
  sed -i.bak '/"dev"/d' versions.json
  rm -f versions.json.bak
  grep -q '"' versions.json || rm -f versions.json
fi

npm run docusaurus docs:version dev
