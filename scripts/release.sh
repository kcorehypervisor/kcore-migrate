#!/usr/bin/env bash
# Publish kcore-migrate archives to a GitHub Release.
# Run from `nix develop` so gh and sha256sum are on PATH.
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"

version="$(tr -d '[:space:]' < VERSION)"
tag="v${version}"

if [[ -f .env ]]; then
	set -a
	# shellcheck disable=SC1091
	source .env
	set +a
fi

dirty() {
	if ! git diff --quiet || ! git diff --cached --quiet; then
		echo "working tree has uncommitted changes" >&2
		exit 1
	fi
	if [[ -n "$(git ls-files --others --exclude-standard)" ]]; then
		echo "working tree has untracked files" >&2
		exit 1
	fi
}

tag_release() {
	dirty
	if git rev-parse -q --verify "refs/tags/${tag}" >/dev/null; then
		if [[ "$(git rev-parse "refs/tags/${tag}^{}")" != "$(git rev-parse HEAD)" ]]; then
			echo "tag ${tag} already points at a different commit" >&2
			exit 1
		fi
	else
		git tag -a "$tag" -m "kcore-migrate ${tag}"
	fi
	git push origin "refs/tags/${tag}"
}

publish() {
	dirty
	if ! git rev-parse -q --verify "refs/tags/${tag}" >/dev/null; then
		echo "tag ${tag} is missing; run make release" >&2
		exit 1
	fi
	if [[ "$(git rev-parse "refs/tags/${tag}^{}")" != "$(git rev-parse HEAD)" ]]; then
		echo "tag ${tag} does not point at HEAD" >&2
		exit 1
	fi
	if [[ ! -f dist/SHA256SUMS ]]; then
		echo "dist/SHA256SUMS is missing; run make dist" >&2
		exit 1
	fi
	local notes=(--generate-notes)
	if [[ -n "${RELEASE_NOTES:-}" ]]; then
		notes=(--notes-file "$RELEASE_NOTES")
	fi
	if gh release view "$tag" >/dev/null 2>&1; then
		gh release upload "$tag" dist/* --clobber
	else
		gh release create "$tag" dist/* "${notes[@]}" --title "$tag"
	fi
}

case "${1:-}" in
release)
	tag_release
	make dist
	publish
	;;
publish)
	publish
	;;
*)
	echo "usage: $0 release|publish" >&2
	exit 2
	;;
esac
