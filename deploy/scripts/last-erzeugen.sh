#!/usr/bin/env bash
# Erzeugt Grundlast, damit das Beobachtungsfenster aussagekräftig ist.
# Für die Vorführung in einem eigenen Terminal laufen lassen.
#
# Aufruf: last-erzeugen.sh [basis-url] [anfragen-pro-sekunde]
set -euo pipefail

BASIS="${1:-http://localhost}"
RATE="${2:-5}"
PAUSE="$(awk -v r="$RATE" 'BEGIN{printf "%.3f", 1/r}')"

echo "Erzeuge etwa $RATE Anfragen/s gegen $BASIS (Abbruch mit Strg+C)"
trap 'echo; echo "beendet"; exit 0' INT

while true; do
	curl -s -o /dev/null "$BASIS/" || true
	curl -s -o /dev/null "$BASIS/api/links" || true
	sleep "$PAUSE"
done
