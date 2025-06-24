#!/bin/bash
set -x
LOCALBIN="$(pwd)/bin"
ENVTEST="setup-envtest"
ENVTEST_VERSION="release-0.19"
PACKAGE="sigs.k8s.io/controller-runtime/tools/setup-envtest"

TARGET="${ENVTEST}-${ENVTEST_VERSION}"
pushd "$LOCALBIN"
if [ ! -f "$TARGET" ]; then
  set -e
  echo "Downloading ${PACKAGE}@${ENVTEST_VERSION}"
  rm -f "$ENVTEST" || true
  GOBIN="$LOCALBIN" go install "${PACKAGE}@${ENVTEST_VERSION}"
  mv "$ENVTEST" "$TARGET"
fi

ls -lha
ln -sf "$TARGET" "$ENVTEST"
popd