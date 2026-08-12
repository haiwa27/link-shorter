package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"time"

	"healthgate/internal/shortener"
	"healthgate/internal/store"
)

type anlegenAnfrage struct {
	Ziel       string `json:"ziel"`
	WunschSlug string `json:"wunschSlug,omitempty"`
}

// linkAnlegen erzeugt einen Kurzlink (Stories P-01 und P-05).
func (h *handler) linkAnlegen(w http.ResponseWriter, r *http.Request) {
	var anfrage anlegenAnfrage
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10)).Decode(&anfrage); err != nil {
		antwortFehler(w, http.StatusBadRequest, "Anfrage ist kein gültiges JSON")
		return
	}

	ziel, err := shortener.PruefeZiel(anfrage.Ziel)
	if err != nil {
		antwortFehler(w, http.StatusBadRequest, err.Error())
		return
	}

	slug := anfrage.WunschSlug
	if slug == "" {
		slug, err = shortener.ErzeugeSlug(7)
		if err != nil {
			h.Log.Error("Slug konnte nicht erzeugt werden", "fehler", err)
			antwortFehler(w, http.StatusInternalServerError, "Slug konnte nicht erzeugt werden")
			return
		}
	} else if err := shortener.PruefeSlug(slug); err != nil {
		antwortFehler(w, http.StatusBadRequest, err.Error())
		return
	}

	link := store.Link{Slug: slug, Ziel: ziel, Erstellt: time.Now().UTC()}
	if err := h.Speicher.Anlegen(r.Context(), link); err != nil {
		if errors.Is(err, store.ErrBelegt) {
			antwortFehler(w, http.StatusConflict, "Slug ist bereits belegt")
			return
		}
		h.Log.Error("Anlegen fehlgeschlagen", "fehler", err)
		antwortFehler(w, http.StatusInternalServerError, "Anlegen fehlgeschlagen")
		return
	}

	antwortJSON(w, http.StatusCreated, link)
}

// linkListe liefert alle Links, neueste zuerst (Story P-03).
func (h *handler) linkListe(w http.ResponseWriter, r *http.Request) {
	liste, err := h.Speicher.Liste(r.Context())
	if err != nil {
		h.Log.Error("Liste fehlgeschlagen", "fehler", err)
		antwortFehler(w, http.StatusInternalServerError, "Liste konnte nicht geladen werden")
		return
	}
	sort.Slice(liste, func(i, j int) bool {
		return liste[i].Erstellt.After(liste[j].Erstellt)
	})
	antwortJSON(w, http.StatusOK, map[string]any{"links": liste})
}

// linkLoeschen entfernt einen Kurzlink (Story P-04).
func (h *handler) linkLoeschen(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if err := h.Speicher.Loeschen(r.Context(), slug); err != nil {
		if errors.Is(err, store.ErrNichtGefunden) {
			antwortFehler(w, http.StatusNotFound, "Link nicht gefunden")
			return
		}
		h.Log.Error("Löschen fehlgeschlagen", "fehler", err)
		antwortFehler(w, http.StatusInternalServerError, "Löschen fehlgeschlagen")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// weiterleiten löst einen Kurzlink auf (Story P-02).
//
// Bewusst 302 und nicht 301: eine dauerhafte Weiterleitung wird vom Browser
// gecacht. Beim Wiederholen der E2E-Tests und beim Zählen der Aufrufe für die
// Metriken wäre das eine stille Fehlerqülle.
func (h *handler) weiterleiten(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")

	link, err := h.Speicher.Holen(r.Context(), slug)
	if err != nil {
		if errors.Is(err, store.ErrNichtGefunden) {
			// Kein Kurzlink: an das Frontend durchreichen, damit Pfade wie
			// /favicon.ico weiterhin bedient werden.
			h.statisch.ServeHTTP(w, r)
			return
		}
		h.Log.Error("Auflösen fehlgeschlagen", "fehler", err, "slug", slug)
		antwortFehler(w, http.StatusInternalServerError, "Auflösen fehlgeschlagen")
		return
	}

	if err := h.Speicher.AufrufZaehlen(r.Context(), slug); err != nil {
		// Zählfehler darf die Weiterleitung nicht verhindern.
		h.Log.Warn("Aufruf konnte nicht gezählt werden", "fehler", err, "slug", slug)
	}

	http.Redirect(w, r, link.Ziel, http.StatusFound)
}
