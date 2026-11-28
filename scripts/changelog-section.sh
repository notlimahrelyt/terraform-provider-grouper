#!/usr/bin/env bash
# Print the CHANGELOG.md section for one version, without its heading.
#
#   scripts/changelog-section.sh 0.2.0
#
# Exits non-zero if the version has no section, so a release cannot go out
# without notes.
set -euo pipefail

version="${1#v}"
changelog="${2:-CHANGELOG.md}"

notes="$(awk -v v="$version" '
  /^## \[/       { if (found) exit; if (index($0, "## [" v "]") == 1) { found = 1; next } }
  /^\[[^]]+\]: / { if (found) exit }
  found          { print }
' "$changelog")"

# Trim leading and trailing blank lines.
notes="$(printf '%s\n' "$notes" | sed -e '/./,$!d')"
notes="$(printf '%s\n' "$notes" | sed -e :a -e '/^\n*$/{$d;N;ba' -e '}')"

if [ -z "$notes" ]; then
  echo "error: no \"## [$version]\" section in $changelog" >&2
  exit 1
fi

printf '%s\n' "$notes"
