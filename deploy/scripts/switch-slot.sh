#!/usr/bin/env bash
# Schaltet den Verkehr auf den angegebenen Slot um (Story R-02).
#
# Aufruf: switch-slot.sh <blue|green>
set -euo pipefail

# Die Slot-Datei wird abwechselnd von einem Menschen und von der Pipeline
# geschrieben, also von zwei verschiedenen Benutzern. Ohne diese Maske legt der
# eine sie mit 600 an und der andere kommt nicht mehr heran -- das Umschalten
# von Hand scheitert dann an einem "Permission denied" (Entscheidung E-024).
umask 002

ZIEL="${1:?Slot fehlt (blue|green)}"
if [ "$ZIEL" != "blue" ] && [ "$ZIEL" != "green" ]; then
	echo "FEHLER: Slot muss blue oder green sein, ist '$ZIEL'" >&2
	exit 1
fi

HIER="$(cd "$(dirname "$0")" && pwd)"
KONF="${KONF_DATEI:-$HIER/../caddy/active-slot.conf}"
ZUSTAND="${ZUSTAND_VERZEICHNIS:-$HIER/../state}"
CADDY_CONTAINER="${CADDY_CONTAINER:-healthgate-prod-caddy-1}"

# Erster Lauf auf einer frischen Maschine: die Datei ist Laufzeitzustand und
# liegt deshalb nicht im Repository. Ohne diesen Zweig legte Docker beim Start
# von Caddy ein Verzeichnis an dieser Stelle an und der Proxy käme nicht hoch.
if [ ! -f "$KONF" ]; then
	echo "Hinweis: $KONF fehlt, wird aus der Vorlage angelegt."
	cp "$KONF.vorlage" "$KONF"
	# cp übernimmt die Rechte der Vorlage; hier zählt, dass die Gruppe
	# schreiben darf.
	chmod 664 "$KONF" 2>/dev/null || true
fi

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
