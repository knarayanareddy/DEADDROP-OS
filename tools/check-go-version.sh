#!/bin/sh
set -eu
MIN="go1.25.11"
HAVE=$(go version | awk '{print $3}')
if [ "$HAVE" != "$MIN" ]; then
  echo "DEADDROP-OS production builds require $MIN exactly; found $HAVE" >&2
  echo "Install https://go.dev/dl/$MIN.linux-amd64.tar.gz or update the pinned release process explicitly." >&2
  exit 1
fi
