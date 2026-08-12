#!/usr/bin/env bash
# Schaltet auf den vorherigen Slot zurück (Stories R-05 und R-07).
#
# Kein Neubau, kein Image-Download: der alte Slot laeuft noch, es wird nur der
# Verkehr zurückgelenkt. Das ist der Grund, weshalb Blue/Green überhaupt
# gewählt wurde -- der Rückweg dauert Sekunden, nicht Minuten.
set -euo pipefail

HIER="$(cd "$(dirname "$0")" && pwd)"
ZUSTAND="${ZUSTAND_VERZEICHNIS:-$HIER/../state}"
GRUND="${1:-nicht angegeben}"

if [ ! -f "$ZUSTAND/vorheriger-slot" ]; then
	echo "FEHLER: kein vorheriger Slot vermerkt, Rollback nicht moeglich." >&2
	exit 1
fi

ZURUECK="$(cat "$ZUSTAND/vorheriger-slot")"
AKTIV="$("$HIER/active-slot.sh")"

echo "==> Rollback: $AKTIV -> $ZURUECK"
echo "    Grund: $GRUND"

"$HIER/switch-slot.sh" "$ZURUECK"
"$HIER/wait-healthy.sh" "${BASIS_URL:-http://localhost}" 3 15

printf '%s\t%s\t%s\t%s\n' "$(date -Iseconds)" "$ZURUECK" "zurueckgerollt" "$GRUND" \
	>> "$ZUSTAND/historie.tsv"

echo "OK: Rollback abgeschlossen, Verkehr laeuft auf $ZURUECK."
