#!/usr/bin/env bash
# Wartet, bis eine Instanz mehrfach hintereinander gesund meldet (Story R-03).
#
# Aufruf: wait-healthy.sh <ziel> [erfolge] [versuche]
#   ziel = Basis-URL          z. B. http://localhost:8081
#   ziel = container:<name>   z. B. container:healthgate-prod-app-green-1
#
# Die zweite Form wird vor dem Umschalten gebraucht: die Produktionsslots
# veröffentlichen bewusst keinen Port auf dem Host, sie sind nur über den
# Reverse Proxy und im internen Netz erreichbar. Geprüft wird trotzdem der
# Container selbst und nicht der Proxy -- sonst prüfte man den alten Slot.
set -euo pipefail

ZIEL="${1:?Ziel fehlt (Basis-URL oder container:<name>)}"
NOETIGE_ERFOLGE="${2:-3}"
MAX_VERSUCHE="${3:-30}"
PAUSE="${WAIT_PAUSE:-2}"

pruefen() {
	case "$ZIEL" in
		container:*)
			docker exec "${ZIEL#container:}" \
				wget -qO- --timeout=3 http://127.0.0.1:8080/healthz 2>/dev/null
			;;
		*)
			curl -fsS --max-time 3 "$ZIEL/healthz" 2>/dev/null
			;;
	esac
}

erfolge=0
for versuch in $(seq 1 "$MAX_VERSUCHE"); do
	if antwort="$(pruefen)"; then
		erfolge=$((erfolge + 1))
		printf '  Versuch %2d: gesund (%d/%d)  %s\n' "$versuch" "$erfolge" "$NOETIGE_ERFOLGE" "$antwort"
		if [ "$erfolge" -ge "$NOETIGE_ERFOLGE" ]; then
			echo "OK: $ZIEL ist bereit"
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

echo "FEHLER: $ZIEL wurde nach $MAX_VERSUCHE Versuchen nicht bereit" >&2
exit 1
