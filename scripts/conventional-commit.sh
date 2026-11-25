#!/usr/bin/env bash
# Checks that commit subjects follow conventional commits, which semantic-release
# reads to compute the next version (.releaserc.json).
#
#   scripts/conventional-commit.sh <message-file>        # commit-msg hook
#   scripts/conventional-commit.sh --range <base> <head> # every commit of a PR (CI)
#
# Allowed: <type>[(scope)][!]: <description>, with type one of the list below.
# Merge commits are skipped, and so are the subjects git writes itself
# ("Merge ...", "Revert \"...\"").
set -euo pipefail

TYPES='feat|fix|perf|refactor|revert|build|docs|test|ci|chore|style'
PATTERN="^(${TYPES})(\([^)]+\))?!?: [^ ].*"

valid() {
	local subject=$1
	[[ $subject =~ $PATTERN ]] || [[ $subject == Merge\ * ]] || [[ $subject == Revert\ \"* ]]
}

usage() {
	cat >&2 <<HELP
Use: <type>(<optional scope>): <description>, e.g. "feat(login): refresh token cookie"
Types: ${TYPES//|/, }
  feat -> minor version; fix, perf, refactor, revert, build -> patch;
  "!" after the type/scope or a "BREAKING CHANGE:" footer -> major;
  docs, test, ci, chore, style -> no release.
HELP
}

if [[ ${1:-} == --range ]]; then
	base=${2:?base commit}
	head=${3:?head commit}
	bad=0
	while IFS= read -r line; do
		sha=${line%% *}
		subject=${line#* }
		if ! valid "$subject"; then
			echo "::error title=Not a conventional commit::${sha} ${subject}"
			bad=1
		fi
	done < <(git log --no-merges --format='%h %s' "${base}..${head}")
	if ((bad)); then
		usage
		echo "Reword them (git rebase -i ${base}) and push again." >&2
		exit 1
	fi
	echo "Every commit follows conventional commits."
	exit 0
fi

file=${1:?usage: conventional-commit.sh <message-file> | --range <base> <head>}
# First line that is neither empty nor a comment
subject=$(grep -v '^#' "$file" | grep -m1 -v '^[[:space:]]*$' || true)
# Folded away by rebase --autosquash; CI (--range) still rejects any left over
[[ $subject =~ ^(fixup|squash|amend)!\  ]] && exit 0
if ! valid "$subject"; then
	echo "Not a conventional commit: \"${subject}\"" >&2
	usage
	exit 1
fi
