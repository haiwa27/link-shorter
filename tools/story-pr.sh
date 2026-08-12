#!/usr/bin/env bash
# Pusht den aktüllen Branch und öffnet einen Pull bzw. Merge Request.
set -euo pipefail

ZWEIG="$(git rev-parse --abbrev-ref HEAD)"
if [ "$ZWEIG" = "main" ]; then
	echo "FEHLER: auf main gibt es keinen PR zu öffnen." >&2
	exit 1
fi

# Story-ID aus dem Branchnamen zurückgewinnen: feature/r-04-... -> R-04
ID="$(printf '%s' "$ZWEIG" | sed -nE 's|^feature/([pqcord])-([0-9]{2})-.*|\1-\2|p' | tr '[:lower:]' '[:upper:]')"
TITEL="$(git log -1 --pretty=%s)"

git push -u origin "$ZWEIG"

if command -v gh >/dev/null 2>&1; then
	gh pr create --base main --head "$ZWEIG" \
		--title "${ID:+[$ID] }$TITEL" \
		--body "Umsetzt: ${ID:-siehe Branchname}"
elif command -v glab >/dev/null 2>&1; then
	glab mr create --source-branch "$ZWEIG" --target-branch main \
		--title "${ID:+[$ID] }$TITEL" \
		--description "Umsetzt: ${ID:-siehe Branchname}" --remove-source-branch
else
	echo ""
	echo "Weder gh noch glab vorhanden. Branch ist gepusht -- PR bitte im Browser öffnen."
fi
