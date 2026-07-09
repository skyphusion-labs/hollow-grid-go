#!/usr/bin/env bash
# Dispatch a rust-choir roll to fleet-chezmoi after a GHCR image push.
# Requires the pushed image to still be present locally (run in the same job as docker push).
set -euo pipefail

image_path="skyphusion-labs/hollow-grid-go"
image_repo="ghcr.io/${image_path}"

if [ -z "${FLEET_DISPATCH_TOKEN:+SET}" ]; then
  echo "::error::FLEET_DISPATCH_TOKEN is unset -- cannot dispatch fleet-chezmoi roll (org secret, visibility all)."
  exit 1
fi

short="$(printf '%s' "${GITHUB_SHA}" | cut -c1-7)"
repo_digest="$(docker inspect --format='{{index .RepoDigests 0}}' "${image_repo}:${GITHUB_SHA}" 2>/dev/null || true)"
digest="${repo_digest#*@}"
if [ -z "$digest" ] || [ "$digest" = "$repo_digest" ]; then
  echo "::error::digest lookup failed for ${image_repo}:${GITHUB_SHA} (is the image still local after push?)"
  exit 1
fi

image="${image_repo}:${short}@${digest}"
payload="$(jq -nc --arg image "$image" --arg sha "$GITHUB_SHA" \
  '{event_type:"rust-choir-roll",client_payload:{image:$image,sha:$sha}}')"

code="$(curl -sS -o /tmp/fleet_dispatch_resp.txt -w '%{http_code}' \
  -X POST \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer ${FLEET_DISPATCH_TOKEN}" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  https://api.github.com/repos/skyphusion-labs/fleet-chezmoi/dispatches \
  -d "$payload")"
echo "repository_dispatch (rust-choir-roll) -> HTTP ${code}"
if [ "$code" != "204" ]; then
  echo "::error::fleet-chezmoi rust-choir-roll dispatch failed (HTTP ${code})."
  cat /tmp/fleet_dispatch_resp.txt || true
  exit 1
fi
echo "rust-choir-roll dispatch accepted (${image})."
