#!/usr/bin/env bash
#
# Fails when a commit, or the PR title, carries a Conventional Commits
# breaking-change marker. release-please turns those into a major bump, and
# EncomPlayer is pinned to 1.x (see AGENTS.md).
#
# Usage: no-breaking.sh <git-revision-range>
# Set PR_TITLE to check a pull request title as well.

set -euo pipefail

range="${1:?usage: no-breaking.sh <git-revision-range>}"
subject_re='^[a-zA-Z]+(\([^)]*\))?!:'
footer_re='^BREAKING[ -]CHANGE:'
status=0

check() {
	local label="$1" message="$2"
	if head -n1 <<<"$message" | grep -Eq "$subject_re" || grep -Eiq "$footer_re" <<<"$message"; then
		echo "::error::$label marks a breaking change. EncomPlayer is pinned to 1.x; use feat: or fix: without '!' or a BREAKING CHANGE footer."
		status=1
	fi
}

if [[ -n "${PR_TITLE:-}" ]]; then
	check "PR title" "$PR_TITLE"
fi

while read -r sha; do
	check "commit ${sha:0:7}" "$(git log -1 --format=%B "$sha")"
done < <(git rev-list "$range")

exit "$status"
