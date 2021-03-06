#!/bin/bash

if [ "$#" -eq 0 ]; then
  VERSION=latest
else
  VERSION=$1
fi

BASEDIR=$(dirname "$0")
PROJECT_DIR="$BASEDIR/.."
CONFIG_DIR="$PROJECT_DIR/config"

RELEASE_MANIFEST="$CONFIG_DIR/release.yaml"

TARGETS=("$CONFIG_DIR/controller/controller.yaml" "$CONFIG_DIR/rbac/role.yaml" "$CONFIG_DIR/rbac/role_binding.yaml" "$CONFIG_DIR/rbac/service_account.yaml" "$CONFIG_DIR/crd/bases")

function append_target(){
  local TARGET="$1"

  if [ "${TARGET: -5}" == ".yaml" ]; then
    cat "$TARGET" >> "$RELEASE_MANIFEST"
    echo "---" >> "$RELEASE_MANIFEST"
  else
    for f in "$TARGET"/*; do
      append_target "$f"
    done
  fi
}

rm -rf "$RELEASE_MANIFEST"

touch "$RELEASE_MANIFEST"

for target in "${TARGETS[@]}"; do
  append_target "$target"
done

sed -i "s/tmaxcloudck\/cd-operator:latest/tmaxcloudck\/cd-operator:$VERSION/g" "$RELEASE_MANIFEST"
