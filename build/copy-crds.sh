#!/bin/bash
# Copyright Contributors to the Open Cluster Management project

KLUSTERLET_CRD_FILE="./vendor/open-cluster-management.io/api/operator/v1/0000_00_operator.open-cluster-management.io_klusterlets.crd.yaml"
CLUSTER_MANAGER_CRD_FILE="./vendor/open-cluster-management.io/api/operator/v1/0000_01_operator.open-cluster-management.io_clustermanagers.crd.yaml"

ALL_FILES=("./pkg/cmd/join/scenario/join/klusterlets.crd.yaml"
           "./pkg/cmd/init/scenario/init/clustermanagers.crd.yaml"
          )

cp "$KLUSTERLET_CRD_FILE" "${ALL_FILES[0]}"
cp "$CLUSTER_MANAGER_CRD_FILE" "${ALL_FILES[1]}"

COMMUNITY_COPY_HEADER_FILE="$PWD/build/copyright-header.txt"

if [ ! -f "$COMMUNITY_COPY_HEADER_FILE" ]; then
  echo "File $COMMUNITY_COPY_HEADER_FILE not found!"
  exit 1
fi
COMMUNITY_COPY_HEADER_STRING=$(cat "$COMMUNITY_COPY_HEADER_FILE")

for FILE in "${ALL_FILES[@]}"
do
  if [ -f "$FILE" ] && ! grep -q "$COMMUNITY_COPY_HEADER_STRING" "$FILE"; then
    echo "# $COMMUNITY_COPY_HEADER_STRING" > tempfile;
    cat "$FILE" >> tempfile;
    mv tempfile "$FILE";
  fi
done
