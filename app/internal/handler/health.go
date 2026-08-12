package handler

import "net/http"

// healthz meldet die Bereitschaft der Instanz (Story O-01).
//
// Wichtig für den Health Gate: dieser Endpunkt beantwortet nur die Frage
// "kann diese Instanz Verkehr annehmen". Ob die ausgelieferte Version fachlich
// korrekt arbeitet, sagt er nicht -- dafür ist das metrikbasierte
// Beobachtungsfenster zuständig (Story R-04).
func (h *handler) healthz(w http.ResponseWriter, r *http.Request) {
	if err := h.Speicher.Pruefen(r.Context()); err != nil {
		h.Log.Warn("Bereitschaftsprüfung fehlgeschlagen", "fehler", err)
		antwortJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "nicht bereit",
			"grund":  err.Error(),
			"slot":   h.Slot,
		})
		return
	}
	antwortJSON(w, http.StatusOK, map[string]string{
		"status":  "bereit",
		"version": h.Version,
		"slot":    h.Slot,
	})
}
