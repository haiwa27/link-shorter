// Paket chaos erzeugt reproduzierbar Fehlerantworten (Story R-06).
//
// Zwei bewusste Entscheidungen:
//
//  1. Die Fehlerrate kommt ausschliesslich aus einer Umgebungsvariable und ist
//     zur Laufzeit nicht umschaltbar. Es gibt also keinen Endpunkt, über den
//     jemand von aussen Fehler auslösen könnte. Für die Vorführung wird
//     eine Version mit gesetzter Rate ausgeliefert.
//
//  2. /healthz ist ausgenommen. Das ist kein Versehen: bliebe der Health-Check
//     ebenfalls rot, würde die Pipeline schon vor dem Umschalten abbrechen und
//     der metrikbasierte Rollback käme nie zum Einsatz. Genau dieser Fall --
//     Prozess lebt und meldet sich gesund, liefert aber fehlerhafte Antworten
//     -- ist der Grund, weshalb ein Health-Check allein nicht ausreicht.
package chaos

import (
	"math/rand/v2"
	"net/http"
)

func Middleware(rate float64, next http.Handler) http.Handler {
	if rate <= 0 {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" || r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}
		if rand.Float64() < rate {
			http.Error(w, "absichtlich erzeugter Fehler", http.StatusInternalServerError)
			return
		}
		next.ServeHTTP(w, r)
	})
}
