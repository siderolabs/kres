#!/usr/bin/env bash

# Manages the GitHub App credentials of the slash-merge workflow (see common.Repository.SlashMerge).
#
# The credentials are stored sops-encrypted to the developers' PGP keys in hack/slash-merge-app.sops.yaml,
# and provisioned to the `slash-merge` environment of each repository by kres.
#
# Usage:
#
#   hack/slash-merge-app.sh encrypt <private-key.pem> [client-id]
#
#     Encrypts the app private key (e.g. after generating a new one to rotate it) into
#     hack/slash-merge-app.sops.yaml, without writing the plaintext to disk.
#
#   GITHUB_TOKEN=... hack/slash-merge-app.sh provision <repository checkout>...
#
#     Decrypts the credentials once, and runs kres in each repository checkout to store them in its
#     `slash-merge` environment. kres checks the credentials against the app installation before storing them.
#     By default `make rekres` is used (ghcr.io/siderolabs/kres:latest, override with KRES_IMAGE),
#     set KRES_BIN to the path of a kres binary to run it directly instead.
#
# Rotating the private key: generate a new key in the app settings, `encrypt` it, `provision` all repositories,
# and delete the old key in the app settings. Removing a recipient from .sops.yaml requires a rotation as well,
# as they might have decrypted the current key already.

set -euo pipefail

DEFAULT_CLIENT_ID="Iv23liSXte1DT1reIQfw"

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SOPS_FILE="hack/slash-merge-app.sops.yaml" # relative to ROOT, to match the .sops.yaml creation rule

function usage {
  sed -n '/^# Usage:/,/^# Rotating/p' "${BASH_SOURCE[0]}" | sed -e 's/^# \{0,1\}//' -e '$d' >&2
  exit 1
}

function encrypt {
  local pem="${1:-}" client_id="${2:-${DEFAULT_CLIENT_ID}}"

  [[ -n "${pem}" ]] || usage

  if ! grep -q -- "-----BEGIN RSA PRIVATE KEY-----" "${pem}"; then
    echo "${pem} doesn't look like a GitHub App private key" >&2
    exit 1
  fi

  cd "${ROOT}"

  # not local: the EXIT trap runs after the function returns
  tmp="$(mktemp "${SOPS_FILE}.XXXXXX")"
  trap 'rm -f "${tmp}"' EXIT

  CLIENT_ID="${client_id}" PRIVATE_KEY="$(cat "${pem}")" \
    yq -n '.client_id = strenv(CLIENT_ID) | .private_key = strenv(PRIVATE_KEY)' |
    sops encrypt --filename-override "${SOPS_FILE}" >"${tmp}"

  mv "${tmp}" "${SOPS_FILE}"

  echo "encrypted ${pem} (client ID ${client_id}) to ${SOPS_FILE}" >&2
}

function provision {
  [[ $# -gt 0 ]] || usage

  if [[ -z "${GITHUB_TOKEN:-}" ]]; then
    echo "GITHUB_TOKEN must be set, kres skips the GitHub API integration without it" >&2
    exit 1
  fi

  # resolve all checkouts before decrypting, so that a typo doesn't cost a YubiKey touch
  local repos=() repo

  for repo in "$@"; do
    repos+=("$(cd "${repo}" && pwd)")
  done

  local plaintext
  plaintext="$(sops decrypt "${ROOT}/${SOPS_FILE}")"

  KRES_SLASH_MERGE_APP_CLIENT_ID="$(yq -r '.client_id' <<<"${plaintext}")"
  KRES_SLASH_MERGE_APP_PRIVATE_KEY="$(yq -r '.private_key' <<<"${plaintext}")"
  unset plaintext

  export KRES_SLASH_MERGE_APP_CLIENT_ID KRES_SLASH_MERGE_APP_PRIVATE_KEY

  for repo in "${repos[@]}"; do
    echo "==> ${repo}" >&2

    if [[ -n "${KRES_BIN:-}" ]]; then
      (cd "${repo}" && "${KRES_BIN}" gen)
    else
      make -C "${repo}" rekres
    fi
  done
}

case "${1:-}" in
  encrypt)
    shift
    encrypt "$@"
    ;;
  provision)
    shift
    provision "$@"
    ;;
  *)
    usage
    ;;
esac
