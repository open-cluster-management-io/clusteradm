#! /bin/bash
# Copyright Contributors to the Open Cluster Management project

set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
repo_dir="$(realpath "${script_dir}/..")"

version_format="vX.Y.Z[-suffix]"

# Check if the release version is provided
if [[ $# -ne 1 ]]; then
  echo "error: a single release version positional argument ${version_format} is required" >&2
  exit 1
fi

RELEASE_VERSION=${1}

# Check if the release version is valid
if [[ ! ${RELEASE_VERSION} =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-.*)?$ ]]; then
  echo "error: invalid release version (expected format: ${version_format}): ${RELEASE_VERSION}" >&2
  exit 1
fi

# Check if the release version is already used
if [[ -n $(git -C "${repo_dir}" tag -l "${RELEASE_VERSION}") ]]; then
  echo "error: release version ${RELEASE_VERSION} already exists" >&2
  exit 1
fi

# Update the version.json file with the new release version
current_default_json=$(jq -r '.default' "${repo_dir}/pkg/version/version.json")
new_default_json=$(
  echo "${current_default_json}" |
    jq -r --arg version "${RELEASE_VERSION}" '.ocm = $version'
)
echo "${new_default_json}" >"${repo_dir}/pkg/version/testdata/current-version.json"

jq --argjson version_bundle "${new_default_json}" \
  --arg version "${RELEASE_VERSION#v}" '
  .default = $version_bundle |
  .[$version] = $version_bundle
  ' "${repo_dir}/pkg/version/version.json" >"${repo_dir}/pkg/version/version.json.tmp"

mv "${repo_dir}/pkg/version/version.json.tmp" "${repo_dir}/pkg/version/version.json"

cd "${repo_dir}"

# Upgrade all Open Cluster Management packages to the latest version
for package in $(
  go list -mod=readonly -m -f '{{if not (or .Indirect .Main)}}{{.Path}}{{end}}' all |
    grep ^open-cluster-management.io
); do
  go get "${package}@latest"
done

# Ensure Kubernetes package consistency
k8s_version=$(go list -m -f '{{ .Version }}' k8s.io/api)
for package in apiextensions-apiserver cli-runtime kubectl; do
  go get "k8s.io/${package}@${k8s_version}"
done

go mod tidy
go mod vendor
