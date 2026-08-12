package handler

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"healthgate/internal/metrics"
	"healthgate/internal/store"
)

func testRouter() http.Handler {
	return NeuerRouter(Abhaengigkeiten{
		Speicher:       store.NeuerSpeicher(),
		Metriken:       metrics.NeueRegistry("test", "testversion"),
		Version:        "testversion",
		Slot:           "test",
		WebVerzeichnis: "./nicht-vorhanden",
		Log:            slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
}

func TestHealthzIstBereit(t *testing.T) {
	w := httptest.NewRecorder()
	testRouter().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("erwartet 200, erhalten %d", w.Code)
	}
	var koerper map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &koerper); err != nil {
		t.Fatalf("Antwort ist kein JSON: %v", err)
	}
	if koerper["status"] != "bereit" {
		t.Errorf("status: erwartet bereit, erhalten %q", koerper["status"])
	}
	if koerper["version"] != "testversion" {
		t.Errorf("version: erwartet testversion, erhalten %q", koerper["version"])
	}
}

func TestLinkAnlegenUndWeiterleiten(t *testing.T) {
	router := testRouter()

	w := httptest.NewRecorder()
	anfrage := httptest.NewRequest(http.MethodPost, "/api/links",
		strings.NewReader(`{"ziel":"https://www.beispiel.de/ziel"}`))
	router.ServeHTTP(w, anfrage)

	if w.Code != http.StatusCreated {
		t.Fatalf("Anlegen: erwartet 201, erhalten %d (%s)", w.Code, w.Body.String())
	}
	var link store.Link
	if err := json.Unmarshal(w.Body.Bytes(), &link); err != nil {
		t.Fatalf("Antwort ist kein JSON: %v", err)
	}
	if link.Slug == "" {
		t.Fatal("Slug fehlt in der Antwort")
	}

	w = httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/"+link.Slug, nil))

	if w.Code != http.StatusFound {
		t.Errorf("Weiterleitung: erwartet 302, erhalten %d", w.Code)
	}
	if ort := w.Header().Get("Location"); ort != "https://www.beispiel.de/ziel" {
		t.Errorf("Location: erhalten %q", ort)
	}
}

func TestLinkAnlegenLehntUngueltigesZielAb(t *testing.T) {
	w := httptest.NewRecorder()
	anfrage := httptest.NewRequest(http.MethodPost, "/api/links",
		strings.NewReader(`{"ziel":"nur-text"}`))
	testRouter().ServeHTTP(w, anfrage)

	if w.Code != http.StatusBadRequest {
		t.Errorf("erwartet 400, erhalten %d", w.Code)
	}
}

func TestWunschSlugDoppeltErgibtKonflikt(t *testing.T) {
	router := testRouter()
	koerper := `{"ziel":"https://beispiel.de","wunschSlug":"meinlink"}`

	for i, erwartet := range []int{http.StatusCreated, http.StatusConflict} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/links", strings.NewReader(koerper)))
		if w.Code != erwartet {
			t.Errorf("Durchlauf %d: erwartet %d, erhalten %d", i, erwartet, w.Code)
		}
	}
}

func TestUnbekannterSlugErgibtNichtGefunden(t *testing.T) {
	w := httptest.NewRecorder()
	testRouter().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/gibtesnicht", nil))

	if w.Code != http.StatusNotFound {
		t.Errorf("erwartet 404, erhalten %d", w.Code)
	}
}

func TestMetrikendpunktLiefertTextformat(t *testing.T) {
	w := httptest.NewRecorder()
	testRouter().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/metrics", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("erwartet 200, erhalten %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "healthgate_build_info") {
		t.Error("Ausgabe enthaelt healthgate_build_info nicht")
	}
}
