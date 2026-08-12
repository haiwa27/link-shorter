// Paket metrics stellt Zähler im Prometheus-Textformat bereit.
//
// Bewusst ohne externe Abhängigkeit umgesetzt, damit das Projektgerüst ohne
// Netzzugriff baut und der erste grüne Build nicht von einem Modul-Download
// abhängt. Wer später echte Histogramme braucht, ersetzt dieses Paket durch
// prometheus/client_golang -- die Schnittstelle nach aussen bleibt gleich.
package metrics

import (
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

type schluessel struct {
	methode string
	route   string
	status  int
}

type Registry struct {
	mu          sync.Mutex
	slot        string
	version     string
	start       time.Time
	anfragen    map[schluessel]uint64
	dauerSumme  map[string]float64
	dauerAnzahl map[string]uint64
}

func NeueRegistry(slot, version string) *Registry {
	return &Registry{
		slot:        slot,
		version:     version,
		start:       time.Now(),
		anfragen:    make(map[schluessel]uint64),
		dauerSumme:  make(map[string]float64),
		dauerAnzahl: make(map[string]uint64),
	}
}

func (r *Registry) Erfassen(methode, route string, status int, dauer time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.anfragen[schluessel{methode: methode, route: route, status: status}]++
	r.dauerSumme[route] += dauer.Seconds()
	r.dauerAnzahl[route]++
}

// Schreiben liefert den Zustand im Prometheus-Textformat aus.
func (r *Registry) Schreiben(w http.ResponseWriter, _ *http.Request) {
	r.mu.Lock()
	anfragen := make([]string, 0, len(r.anfragen))
	for k, v := range r.anfragen {
		anfragen = append(anfragen, fmt.Sprintf(
			"healthgate_http_requests_total{method=%q,route=%q,status=%q,slot=%q} %d",
			k.methode, k.route, strconv.Itoa(k.status), r.slot, v))
	}
	dauer := make([]string, 0, len(r.dauerSumme)*2)
	for route, summe := range r.dauerSumme {
		dauer = append(dauer, fmt.Sprintf(
			"healthgate_http_request_duration_seconds_sum{route=%q,slot=%q} %g",
			route, r.slot, summe))
		dauer = append(dauer, fmt.Sprintf(
			"healthgate_http_request_duration_seconds_count{route=%q,slot=%q} %d",
			route, r.slot, r.dauerAnzahl[route]))
	}
	laufzeit := time.Since(r.start).Seconds()
	slot, version := r.slot, r.version
	r.mu.Unlock()

	// Sortiert ausgeben, damit die Reihenfolge stabil und diffbar ist.
	slices.Sort(anfragen)
	slices.Sort(dauer)

	var b strings.Builder
	b.WriteString("# HELP healthgate_http_requests_total Anzahl der HTTP-Anfragen\n")
	b.WriteString("# TYPE healthgate_http_requests_total counter\n")
	for _, zeile := range anfragen {
		b.WriteString(zeile + "\n")
	}
	b.WriteString("# HELP healthgate_http_request_duration_seconds_sum Summe der Antwortzeiten\n")
	b.WriteString("# TYPE healthgate_http_request_duration_seconds_sum counter\n")
	b.WriteString("# HELP healthgate_http_request_duration_seconds_count Anzahl gemessener Antworten\n")
	b.WriteString("# TYPE healthgate_http_request_duration_seconds_count counter\n")
	for _, zeile := range dauer {
		b.WriteString(zeile + "\n")
	}
	b.WriteString("# HELP healthgate_build_info Ausgelieferte Version je Slot\n")
	b.WriteString("# TYPE healthgate_build_info gauge\n")
	b.WriteString(fmt.Sprintf("healthgate_build_info{version=%q,slot=%q} 1\n", version, slot))
	b.WriteString("# HELP healthgate_uptime_seconds Laufzeit des Prozesses\n")
	b.WriteString("# TYPE healthgate_uptime_seconds gauge\n")
	b.WriteString(fmt.Sprintf("healthgate_uptime_seconds{slot=%q} %g\n", slot, laufzeit))

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
}

// Route bildet einen Pfad auf eine begrenzte Menge von Bezeichnern ab.
//
// Wichtig: niemals den rohen Pfad als Label verwenden. Jeder Slug würde eine
// eigene Zeitreihe erzeugen und Prometheus binnen Tagen unbenutzbar machen
// (Kardinalitätsexplosion).
func Route(pfad string) string {
	switch {
	case pfad == "/":
		return "/"
	case pfad == "/healthz":
		return "/healthz"
	case pfad == "/metrics":
		return "/metrics"
	case strings.HasPrefix(pfad, "/api/links"):
		return "/api/links"
	case strings.HasPrefix(pfad, "/assets/"):
		return "/assets"
	default:
		return "/:slug"
	}
}

type schreiber struct {
	http.ResponseWriter
	status      int
	geschrieben bool
}

func (s *schreiber) WriteHeader(code int) {
	if !s.geschrieben {
		s.status = code
		s.geschrieben = true
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *schreiber) Write(b []byte) (int, error) {
	if !s.geschrieben {
		s.status = http.StatusOK
		s.geschrieben = true
	}
	return s.ResponseWriter.Write(b)
}

// Middleware zählt jede Anfrage. /metrics selbst wird nicht mitgezählt,
// sonst verfälschen die Prometheus-Abfragen die eigene Statistik.
func Middleware(r *Registry, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/metrics" {
			next.ServeHTTP(w, req)
			return
		}
		beginn := time.Now()
		s := &schreiber{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(s, req)
		r.Erfassen(req.Method, Route(req.URL.Path), s.status, time.Since(beginn))
	})
}
