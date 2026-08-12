#!/usr/bin/env bash
# Gibt den Slot aus, der aktüll Verkehr bekommt: blue oder green.
set -euo pipefail

KONF="${KONF_DATEI:-$(dirname "$0")/../caddy/active-slot.conf}"

if [ ! -f "$KONF" ]; then
	echo "FEHLER: $KONF fehlt" >&2
	exit 1
fi

if grep -q 'app-green' "$KONF"; then
	echo green
elif grep -q 'app-blue' "$KONF"; then
	echo blue
else
	echo "FEHLER: kein Slot in $KONF erkennbar" >&2
	exit 1
fi
