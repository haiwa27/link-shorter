#!/usr/bin/env bash
# Legt einen Branch für eine Story an.
#
# Aufruf: tools/story-start.sh R-04 "Beobachtungsfenster nach dem Umschalten"
# Ergebnis: Branch feature/r-04-beobachtungsfenster-nach-dem-umschalten
set -euo pipefail

ID="${1:?Story-ID fehlt, z. B. R-04}"
TITEL="${2:?Titel fehlt}"

if ! printf '%s' "$ID" | grep -qE '^[PQCORD]-[0-9]{2}$'; then
	echo "FEHLER: Story-ID muss die Form P-01, Q-03, R-04 haben." >&2
	exit 1
fi

# Branchnamen bleiben bewusst bei ASCII: Umlaute in Refs führen je nach
# Werkzeug und Dateisystem zu Überraschungen. Im Titel und in der
# Commit-Nachricht werden weiterhin echte Umlaute geschrieben.
kurz="$(printf '%s' "$TITEL" |
	tr '[:upper:]' '[:lower:]' |
	sed -e 's/ä/ae/g' -e 's/ö/oe/g' -e 's/ü/ue/g' -e 's/ß/ss/g' |
	sed -e 's/[^a-z0-9]\+/-/g' -e 's/^-//' -e 's/-$//' |
	cut -c1-50 | sed 's/-$//')"

ZWEIG="feature/$(printf '%s' "$ID" | tr '[:upper:]' '[:lower:]')-$kurz"

if [ -n "$(git status --porcelain)" ]; then
	echo "FEHLER: Arbeitsverzeichnis ist nicht sauber. Erst committen oder stashen." >&2
	git status --short
	exit 1
fi

git checkout main
git pull --ff-only origin main 2>/dev/null || echo "Hinweis: kein Remote erreichbar, main bleibt lokal."
git checkout -b "$ZWEIG"

echo ""
echo "Branch angelegt: $ZWEIG"
echo "Fertig? Dann: tools/story-pr.sh"
