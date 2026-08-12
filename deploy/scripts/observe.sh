#!/usr/bin/env bash
# Beobachtungsfenster nach dem Umschalten -- das Herzstück des Projekts
# (Story R-04).
#
# Fragt wiederholt die Fehlerrate des angegebenen Slots bei Prometheus ab.
# Exitcode 0 = unauffällig, Exitcode 1 = Grenzwert verletzt.
# Der Aufrufer (Jenkins) löst daraufhin den Rollback aus.
#
# Aufruf: observe.sh <blue|green>
set -euo pipefail

SLOT="${1:?Slot fehlt (blue|green)}"
DAUER="${OBSERVE_DAUER:-120}"
INTERVALL="${OBSERVE_INTERVALL:-10}"
SCHWELLE="${OBSERVE_SCHWELLE:-0.05}"
PROM="${PROMETHEUS_URL:-http://localhost:9090}"
MIN_ANFRAGEN="${OBSERVE_MIN_ANFRAGEN:-5}"

command -v jq >/dev/null || { echo "FEHLER: jq wird benoetigt" >&2; exit 2; }

# clamp_min verhindert eine Division durch Null. Ohne Verkehr ergibt die
# Abfrage damit 0 und nicht NaN.
ABFRAGE_RATE="sum(rate(healthgate_http_requests_total{slot=\"$SLOT\",status=~\"5..\"}[1m])) / clamp_min(sum(rate(healthgate_http_requests_total{slot=\"$SLOT\"}[1m])), 0.001)"
ABFRAGE_LAST="sum(increase(healthgate_http_requests_total{slot=\"$SLOT\"}[1m]))"

abfragen() {
	local ausdruck="$1"
	curl -fsSG --max-time 5 "$PROM/api/v1/query" \
		--data-urlencode "query=$ausdruck" |
		jq -r '.data.result[0].value[1] // "0"'
}

echo "==> Beobachtungsfenster fuer Slot '$SLOT'"
echo "    Dauer $DAUER s, Abstand $INTERVALL s, Grenzwert $SCHWELLE, Prometheus $PROM"
printf '\n    %-9s %-14s %-12s %s\n' "Sekunde" "Anfragen/min" "Fehleranteil" "Bewertung"

verstrichen=0
gesehene_last=0

while [ "$verstrichen" -lt "$DAUER" ]; do
	rate="$(abfragen "$ABFRAGE_RATE" || echo "0")"
	last="$(abfragen "$ABFRAGE_LAST" || echo "0")"
	gesehene_last="$(awk -v a="$gesehene_last" -v b="$last" 'BEGIN{print (b>a)?b:a}')"

	if awk -v r="$rate" -v s="$SCHWELLE" 'BEGIN{exit !(r>s)}'; then
		bewertung="VERLETZT"
	else
		bewertung="ok"
	fi

	printf '    %-9s %-14.1f %-12.4f %s\n' "$verstrichen" "$last" "$rate" "$bewertung"

	if [ "$bewertung" = "VERLETZT" ]; then
		echo ""
		echo "FEHLER: Fehleranteil $rate liegt ueber dem Grenzwert $SCHWELLE." >&2
		echo "        Rollback wird ausgeloest." >&2
		exit 1
	fi

	sleep "$INTERVALL"
	verstrichen=$((verstrichen + INTERVALL))
done

# Ohne Verkehr sagt ein grünes Fenster nichts aus. Das offen zu melden ist
# ehrlicher, als ein Deployment stillschweigend zu bestätigen.
if awk -v l="$gesehene_last" -v m="$MIN_ANFRAGEN" 'BEGIN{exit !(l<m)}'; then
	echo ""
	echo "WARNUNG: nur $gesehene_last Anfragen pro Minute gesehen (Mindestwert $MIN_ANFRAGEN)."
	echo "         Das Fenster ist aussagelos. Fuer die Vorfuehrung Last erzeugen,"
	echo "         z.B.: while true; do curl -s http://localhost/ >/dev/null; sleep 0.2; done"
fi

echo ""
echo "OK: Slot '$SLOT' ueber $DAUER s unauffaellig."
