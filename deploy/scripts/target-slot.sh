#!/usr/bin/env bash
# Gibt den Slot aus, der als nächstes bespielt wird (den gerade untätigen).
set -euo pipefail

AKTIV="$("$(dirname "$0")/active-slot.sh")"
if [ "$AKTIV" = "blue" ]; then
	echo green
else
	echo blue
fi
