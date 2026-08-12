#!/usr/bin/env bash
# Setzt ein Release: Version berechnen, Changelog erzeugen, Tag setzen.
#
# Aufruf: tools/release.sh <major|minor|patch>
#
# Ein Release ist hier ein annotierter Tag auf main, kein Branch. Begründung
# steht in docs/git-konventionen.md.
set -euo pipefail

ART="${1:?major, minor oder patch angeben}"
case "$ART" in major|minor|patch) ;; *) echo "FEHLER: major, minor oder patch" >&2; exit 1 ;; esac

ZWEIG="$(git rev-parse --abbrev-ref HEAD)"
[ "$ZWEIG" = "main" ] || { echo "FEHLER: Releases werden nur auf main gesetzt (aktuell: $ZWEIG)." >&2; exit 1; }
[ -z "$(git status --porcelain)" ] || { echo "FEHLER: Arbeitsverzeichnis ist nicht sauber." >&2; exit 1; }

if command -v go >/dev/null 2>&1 && [ -d app ]; then
	echo "==> Tests"
	(cd app && go test ./... >/dev/null) || { echo "FEHLER: Tests sind rot, kein Release." >&2; exit 1; }
	echo "    grün"
fi

LETZTER="$(git describe --tags --abbrev=0 2>/dev/null || echo "")"
if [ -z "$LETZTER" ]; then
	MAJOR=0; MINOR=0; PATCH=0
	BEREICH="$(git rev-list --max-parents=0 HEAD | tail -1)..HEAD"
	echo "==> Erstes Release, alle Commits werden berücksichtigt"
else
	IFS='.' read -r MAJOR MINOR PATCH <<< "${LETZTER#v}"
	BEREICH="$LETZTER..HEAD"
	echo "==> Letztes Release: $LETZTER"
fi

case "$ART" in
	major) MAJOR=$((MAJOR + 1)); MINOR=0; PATCH=0 ;;
	minor) MINOR=$((MINOR + 1)); PATCH=0 ;;
	patch) PATCH=$((PATCH + 1)) ;;
esac
NEU="v$MAJOR.$MINOR.$PATCH"

if [ -z "$(git log --oneline "$BEREICH" 2>/dev/null)" ]; then
	echo "FEHLER: keine Commits seit $LETZTER, nichts zu veröffentlichen." >&2
	exit 1
fi

# --- Changelog aus den Conventional Commits zusammenstellen -----------------
ABSCHNITT="$(mktemp)"
{
	printf '## %s -- %s\n\n' "$NEU" "$(date +%Y-%m-%d)"

	sammeln() {
		local praefix="$1" titel="$2" zeilen
		zeilen="$(git log "$BEREICH" --no-merges --pretty=format:'%s|%h' |
			grep -E "^$praefix(\([a-z0-9-]+\))?!?: " || true)"
		[ -z "$zeilen" ] && return 0
		printf '### %s\n\n' "$titel"
		printf '%s\n' "$zeilen" | while IFS='|' read -r betreff kurz; do
			text="$(printf '%s' "$betreff" | sed -E "s/^$praefix(\([a-z0-9-]+\))?!?: //")"
			bereich="$(printf '%s' "$betreff" | sed -nE 's/^[a-z]+\(([a-z0-9-]+)\).*/\1/p')"
			if [ -n "$bereich" ]; then
				printf -- '- **%s:** %s (%s)\n' "$bereich" "$text" "$kurz"
			else
				printf -- '- %s (%s)\n' "$text" "$kurz"
			fi
		done
		printf '\n'
	}

	sammeln 'feat'     'Neu'
	sammeln 'fix'      'Behoben'
	sammeln 'perf'     'Verbessert'
	sammeln 'refactor' 'Umgebaut'
	sammeln 'docs'     'Dokumentation'
	sammeln 'ci|build' 'Pipeline und Build'
	sammeln 'test'     'Tests'
} > "$ABSCHNITT"

echo ""
cat "$ABSCHNITT"
printf 'Release %s setzen? [j/N] ' "$NEU"
read -r antwort
case "$antwort" in j|J|y|Y) ;; *) echo "Abgebrochen."; rm -f "$ABSCHNITT"; exit 0 ;; esac

# --- Tag auf den aktüllen Stand von main ----------------------------------
git tag -a "$NEU" -F "$ABSCHNITT"
git push --no-verify origin "$NEU"
echo "==> Tag $NEU gesetzt und gepusht"

# --- Changelog über einen Branch nach main, nicht direkt -------------------
NEUES="$(mktemp)"
{
	echo "# Changelog"
	echo ""
	echo "Erzeugt von tools/release.sh aus den Conventional Commits."
	echo ""
	cat "$ABSCHNITT"
	if [ -f CHANGELOG.md ]; then
		tail -n +5 CHANGELOG.md
	fi
} > "$NEUES"
mv "$NEUES" CHANGELOG.md
rm -f "$ABSCHNITT"

RZWEIG="chore/release-$NEU"
git checkout -q -b "$RZWEIG"
git add CHANGELOG.md
git commit -q -m "chore(release): Changelog für $NEU ergänzen"
git push -q -u origin "$RZWEIG"

if command -v gh >/dev/null 2>&1; then
	gh pr create --base main --head "$RZWEIG" \
		--title "chore(release): Changelog für $NEU" \
		--body "Automatisch erzeugt von tools/release.sh. Tag $NEU ist bereits gesetzt."
elif command -v glab >/dev/null 2>&1; then
	glab mr create --source-branch "$RZWEIG" --target-branch main \
		--title "chore(release): Changelog für $NEU" \
		--description "Automatisch erzeugt von tools/release.sh. Tag $NEU ist bereits gesetzt." \
		--remove-source-branch
else
	echo "Branch $RZWEIG gepusht -- Pull Request bitte im Browser öffnen."
fi

echo ""
echo "Fertig: $NEU ist getaggt, Changelog liegt als PR bereit."
