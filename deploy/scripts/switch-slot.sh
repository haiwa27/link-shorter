#!/usr/bin/env bash
# Schaltet den Verkehr auf den angegebenen Slot um (Story R-02).
#
# Aufruf: switch-slot.sh <blue|green>
set -euo pipefail

ZIEL="${1:?Slot fehlt (blue|green)}"
if [ "$ZIEL" != "blue" ] && [ "$ZIEL" != "green" ]; then
	echo "FEHLER: Slot muss blue oder green sein, ist '$ZIEL'" >&2
	exit 1
fi

HIER="$(cd "$(dirname "$0")" && pwd)"
KONF="${KONF_DATEI:-$HIER/../caddy/active-slot.conf}"
ZUSTAND="${ZUSTAND_VERZEICHNIS:-$HIER/../state}"
CADDY_CONTAINER="${CADDY_CONTAINER:-healthgate-prod-caddy-1}"

VORHER="$("$HIER/active-slot.sh")"
if [ "$VORHER" = "$ZIEL" ]; then
	echo "Hinweis: $ZIEL bekommt bereits Verkehr, nichts zu tun."
	exit 0
fi

mkdir -p "$ZUSTAND"
echo "$VORHER" > "$ZUSTAND/vorheriger-slot"

# Erst schreiben, dann neu laden. Caddy übernimmt die Konfiguration über die
# Admin-Schnittstelle ohne bestehende Verbindungen abzubrechen; dadurch bleibt
# das Umschalten für laufende Anfragen unbemerkt.
cat > "$KONF" <<KONFIG
reverse_proxy app-$ZIEL:8080 {
	health_uri /healthz
	health_interval 5s
	health_timeout 2s
}
KONFIG

echo "==> Konfiguration neu laden (Caddy)"
docker exec "$CADDY_CONTAINER" caddy reload --config /etc/caddy/Caddyfile

NACHHER="$("$HIER/active-slot.sh")"
if [ "$NACHHER" != "$ZIEL" ]; then
	echo "FEHLER: Umschalten nicht wirksam, aktiv ist '$NACHHER'" >&2
	exit 1
fi

printf '%s\t%s\t%s\n' "$(date -Iseconds)" "$ZIEL" "umgeschaltet" >> "$ZUSTAND/historie.tsv"
echo "OK: Verkehr laeuft jetzt auf $ZIEL (vorher $VORHER)"
