#!/usr/bin/env bash
# Wartet, bis eine Instanz mehrfach hintereinander gesund meldet (Story R-03).
#
# Aufruf: wait-healthy.sh <basis-url> [erfolge] [versuche]
# Beispiel: wait-healthy.sh http://localhost:8080 3 30
set -euo pipefail

BASIS="${1:?Basis-URL fehlt}"
NOETIGE_ERFOLGE="${2:-3}"
MAX_VERSUCHE="${3:-30}"
PAUSE="${WAIT_PAUSE:-2}"

erfolge=0
for versuch in $(seq 1 "$MAX_VERSUCHE"); do
	if antwort="$(curl -fsS --max-time 3 "$BASIS/healthz" 2>/dev/null)"; then
		erfolge=$((erfolge + 1))
		printf '  Versuch %2d: gesund (%d/%d)  %s\n' "$versuch" "$erfolge" "$NOETIGE_ERFOLGE" "$antwort"
		if [ "$erfolge" -ge "$NOETIGE_ERFOLGE" ]; then
			echo "OK: $BASIS ist bereit"
			exit 0
		fi
	else
		# Aufeinanderfolgende Erfolge, nicht Erfolge insgesamt: eine Instanz,
		# die zwischen gesund und krank pendelt, gilt nicht als bereit.
		erfolge=0
		printf '  Versuch %2d: noch nicht bereit\n' "$versuch"
	fi
	sleep "$PAUSE"
done

echo "FEHLER: $BASIS wurde nach $MAX_VERSUCHE Versuchen nicht bereit" >&2
exit 1
