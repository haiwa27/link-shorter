package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRouteBegrenztKardinalitaet(t *testing.T) {
	faelle := map[string]string{
		"/":                 "/",
		"/healthz":          "/healthz",
		"/metrics":          "/metrics",
		"/api/links":        "/api/links",
		"/api/links/abc123": "/api/links",
		"/assets/index.js":  "/assets",
		"/abc123":           "/:slug",
		"/xyz789":           "/:slug",
	}
	for pfad, erwartet := range faelle {
		if erhalten := Route(pfad); erhalten != erwartet {
			t.Errorf("Route(%q): erwartet %q, erhalten %q", pfad, erwartet, erhalten)
		}
	}
}

func TestSchreibenEnthaeltZaehler(t *testing.T) {
	r := NeueRegistry("blue", "abc1234")
	r.Erfassen(http.MethodGet, "/:slug", http.StatusFound, 12*time.Millisecond)
	r.Erfassen(http.MethodGet, "/:slug", http.StatusInternalServerError, 3*time.Millisecond)

	w := httptest.NewRecorder()
	r.Schreiben(w, httptest.NewRequest(http.MethodGet, "/metrics", nil))

	koerper := w.Body.String()
	for _, teil := range []string{
		`healthgate_http_requests_total{method="GET",route="/:slug",status="302",slot="blue"} 1`,
		`healthgate_http_requests_total{method="GET",route="/:slug",status="500",slot="blue"} 1`,
		`healthgate_build_info{version="abc1234",slot="blue"} 1`,
	} {
		if !strings.Contains(koerper, teil) {
			t.Errorf("Ausgabe enthaelt nicht: %s\n\nvollstaendig:\n%s", teil, koerper)
		}
	}
}

func TestMiddlewareZaehltNichtDenMetrikPfad(t *testing.T) {
	r := NeueRegistry("blue", "dev")
	h := Middleware(r, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/metrics", nil))

	if len(r.anfragen) != 0 {
		t.Errorf("erwartet: keine Zaehlung fuer /metrics, erhalten %d Eintraege", len(r.anfragen))
	}
}
