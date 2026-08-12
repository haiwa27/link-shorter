package chaos

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRateNullReichtDurch(t *testing.T) {
	h := Middleware(0, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/abc", nil))

	if w.Code != http.StatusOK {
		t.Errorf("erwartet 200, erhalten %d", w.Code)
	}
}

func TestHealthzBleibtVerschont(t *testing.T) {
	h := Middleware(1.0, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if w.Code != http.StatusOK {
		t.Errorf("/healthz muss auch bei Rate 1.0 gesund bleiben, erhalten %d", w.Code)
	}
}

func TestRateEinsSchlaegtImmerFehl(t *testing.T) {
	h := Middleware(1.0, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	for i := 0; i < 20; i++ {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/abc", nil))
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("Durchlauf %d: erwartet 500, erhalten %d", i, w.Code)
		}
	}
}
