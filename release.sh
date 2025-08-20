#!/usr/bin/env bash
set -euo pipefail

### --- CONFIG --- ###
NAME="newvm"                 # provider short name
NAMESPACE="newvmcloud"       # registry namespace
VERSION="${1:-}"             # e.g. 1.0.3  (no leading 'v')
TAG="v${VERSION}"
PROTOCOLS='["6.0"]'          # If you use Plugin Framework that requires v6, set '["6.0"]'
GPG_KEY_ID="${GPG_KEY_ID:-}" # optional; if set, gpg --local-user will be used

# Build matrix (add/remove as needed)
PLATFORMS=(
  "darwin_amd64"
  "darwin_arm64"
  "freebsd_amd64"
  "freebsd_arm64"
  "freebsd_arm"
  "freebsd_386"
  "linux_amd64"
  "linux_arm64"
  "linux_arm"
  "linux_386"
  "windows_amd64"
  "windows_arm64"
  "windows_arm"
  "windows_386"
)

### --- CHECKS --- ###
if [[ -z "${VERSION}" ]]; then
  echo "Usage: $0 <version>    (example: $0 1.0.3)"
  exit 1
fi

need() { command -v "$1" >/dev/null 2>&1 || { echo "Missing tool: $1"; exit 1; }; }
need go
need zip
need gpg
need jq
# macOS has 'shasum', most Linux images have 'sha256sum'
if command -v shasum >/dev/null 2>&1; then
  SHACMD='shasum -a 256'
elif command -v sha256sum >/dev/null 2>&1; then
  SHACMD='sha256sum'
else
  echo "Missing shasum/sha256sum"; exit 1
fi

### --- CLEAN OUTPUT DIR --- ###
OUTDIR="build/v${VERSION}"
rm -rf "${OUTDIR}"
mkdir -p "${OUTDIR}"

### --- BUILD & ZIP PER PLATFORM --- ###
# Binary name INSIDE the zip must be exactly terraform-provider-<name>_v<version>[.exe]
for p in "${PLATFORMS[@]}"; do
  GOOS="${p%_*}"
  GOARCH="${p#*_}"
  EXT=""
  [[ "${GOOS}" == "windows" ]] && EXT=".exe"

  BIN="terraform-provider-${NAME}_v${VERSION}${EXT}"
  ZIP="terraform-provider-${NAME}_${VERSION}_${GOOS}_${GOARCH}.zip"

  echo "==> Building ${GOOS}/${GOARCH}"
  # Static build (no external DLLs/so) and smaller binary
  CGO_ENABLED=0 GOOS="${GOOS}" GOARCH="${GOARCH}" \
    go build -ldflags "-s -w" -o "${OUTDIR}/${BIN}"

  # Zip exactly the single file at top level
  ( cd "${OUTDIR}" && zip -q "${ZIP}" "${BIN}" && rm -f "${BIN}" )

done

### --- MANIFEST --- ###
echo "==> Creating manifest"
MANIFEST="${OUTDIR}/terraform-provider-${NAME}_${VERSION}_manifest.json"

# Build platforms JSON array with shasums
PLAT_JSON="[]"
for p in "${PLATFORMS[@]}"; do
  GOOS="${p%_*}"
  GOARCH="${p#*_}"
  ZIP="terraform-provider-${NAME}_${VERSION}_${GOOS}_${GOARCH}.zip"
  SHASUM="$(${SHACMD} "${OUTDIR}/${ZIP}" | awk '{print $1}')"

  PLAT_JSON="$(jq -c \
    --arg os "${GOOS}" \
    --arg arch "${GOARCH}" \
    --arg filename "${ZIP}" \
    --arg shasum "${SHASUM}" \
    '. + [ {os:$os, arch:$arch, filename:$filename, shasum:$shasum} ]' \
    <<< "${PLAT_JSON}")"
done
echo "    ...built platforms JSON"

# Minimal v1 manifest w/ protocols + platforms; write compact JSON w/o trailing newline
MANIFEST_JSON="$(jq -n -c \
  --argjson protocols "${PROTOCOLS}" \
  --argjson platforms "${PLAT_JSON}" \
  '{version:1, metadata:{protocol_versions:$protocols}, platforms:$platforms}')"
echo "    ...built manifest JSON"
printf "%s" "${MANIFEST_JSON}" > "${MANIFEST}"

### --- SHA256SUMS (ZIPs + MANIFEST) --- ###
echo "==> Generating SHA256SUMS"
pushd "${OUTDIR}" >/dev/null
  # Strictly include all zips + the manifest
  ${SHACMD} terraform-provider-${NAME}_${VERSION}_*.zip > "terraform-provider-${NAME}_${VERSION}_SHA256SUMS"
  ${SHACMD} "terraform-provider-${NAME}_${VERSION}_manifest.json" >> "terraform-provider-${NAME}_${VERSION}_SHA256SUMS"

  echo "==> Signing SHA256SUMS"
  if [[ -n "${GPG_KEY_ID}" ]]; then
    gpg --local-user "${GPG_KEY_ID}" --detach-sign "terraform-provider-${NAME}_${VERSION}_SHA256SUMS"
  else
    gpg --detach-sign "terraform-provider-${NAME}_${VERSION}_SHA256SUMS"
  fi
popd >/dev/null

### --- CREATE/UPDATE GITHUB RELEASE & UPLOAD --- ###
if command -v gh >/dev/null 2>&1; then
  echo "==> Ensuring Git tag ${TAG} exists"
  if ! gh release view "${TAG}" >/dev/null 2>&1; then
    # Create tag and release (lightweight tag; for annotated use git tag -a)
    if ! git rev-parse "refs/tags/${TAG}" >/dev/null 2>&1; then
      git tag "${TAG}"
      git push github "${TAG}"
    fi
    echo "==> Creating GitHub release ${TAG}"
    gh release create "${TAG}" \
      --title "${TAG}" \
      --notes "Automated release for ${NAME} ${TAG}"
  else
    echo "==> GitHub release ${TAG} already exists"
  fi

  echo "==> Uploading assets"
  gh release upload "${TAG}" \
    ${OUTDIR}/terraform-provider-${NAME}_${VERSION}_*.zip \
    ${OUTDIR}/terraform-provider-${NAME}_${VERSION}_SHA256SUMS \
    ${OUTDIR}/terraform-provider-${NAME}_${VERSION}_SHA256SUMS.sig \
    ${OUTDIR}/terraform-provider-${NAME}_${VERSION}_manifest.json \
    --clobber
fi

echo "Done. Artifacts are in: ${OUTDIR}"
