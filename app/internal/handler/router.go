// Paket handler bindet die HTTP-Endpunkte an die Fachlogik.
package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"healthgate/internal/metrics"
	"healthgate/internal/store"
)

type Abhaengigkeiten struct {
	Speicher       store.Speicher
	Metriken       *metrics.Registry
	Version        string
	Slot           string
	WebVerzeichnis string
	Log            *slog.Logger
}

type handler struct {
	Abhaengigkeiten
	statisch http.Handler
}

func NeuerRouter(d Abhaengigkeiten) http.Handler {
	h := &handler{
		Abhaengigkeiten: d,
		statisch:        http.FileServer(http.Dir(d.WebVerzeichnis)),
	}

	mux := http.NewServeMux()

	// Betrieb
	mux.HandleFunc("GET /healthz", h.healthz)
	mux.HandleFunc("GET /metrics", d.Metriken.Schreiben)

	// API
	mux.HandleFunc("POST /api/links", h.linkAnlegen)
	mux.HandleFunc("GET /api/links", h.linkListe)
	mux.HandleFunc("DELETE /api/links/{slug}", h.linkLoeschen)

	// Frontend. "/" ist der Auffangpfad für Assets und index.html,
	// "/{slug}" ist spezifischer und greift daher für einsegmentige Pfade.
	mux.Handle("GET /", h.statisch)
	mux.HandleFunc("GET /{slug}", h.weiterleiten)

	return mux
}

func antwortJSON(w http.ResponseWriter, status int, koerper any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if koerper != nil {
		_ = json.NewEncoder(w).Encode(koerper)
	}
}

func antwortFehler(w http.ResponseWriter, status int, meldung string) {
	antwortJSON(w, status, map[string]string{"fehler": meldung})
}
