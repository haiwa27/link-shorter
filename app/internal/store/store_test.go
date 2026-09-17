package store

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// Der In-Memory-Speicher ist der Rückfallweg für den Start ohne Datenbank.
// Ungetestet wäre er genau das, was im Vortrag niemand erklären will: Code,
// der im lokalen Betrieb läuft und dessen Verhalten niemand geprüft hat.

func neuerTestSpeicher(t *testing.T) Speicher {
	t.Helper()
	return NeuerSpeicher()
}

func testLink(slug string) Link {
	return Link{
		Slug:     slug,
		Ziel:     "https://beispiel.de/" + slug,
		Erstellt: time.Now().UTC(),
	}
}

func TestSpeicherAnlegenUndHolen(t *testing.T) {
	s := neuerTestSpeicher(t)
	ctx := context.Background()
	link := testLink("abc1234")

	if err := s.Anlegen(ctx, link); err != nil {
		t.Fatalf("Anlegen: %v", err)
	}

	geholt, err := s.Holen(ctx, link.Slug)
	if err != nil {
		t.Fatalf("Holen: %v", err)
	}
	if geholt.Ziel != link.Ziel {
		t.Errorf("Ziel: erwartet %q, erhalten %q", link.Ziel, geholt.Ziel)
	}
}

func TestSpeicherDoppelterSlugErgibtBelegt(t *testing.T) {
	s := neuerTestSpeicher(t)
	ctx := context.Background()
	link := testLink("doppelt")

	if err := s.Anlegen(ctx, link); err != nil {
		t.Fatalf("Anlegen: %v", err)
	}
	if err := s.Anlegen(ctx, link); !errors.Is(err, ErrBelegt) {
		t.Fatalf("erwartet ErrBelegt, erhalten %v", err)
	}
}

func TestSpeicherHolenUnbekannt(t *testing.T) {
	s := neuerTestSpeicher(t)

	if _, err := s.Holen(context.Background(), "gibtsnicht"); !errors.Is(err, ErrNichtGefunden) {
		t.Fatalf("erwartet ErrNichtGefunden, erhalten %v", err)
	}
}

func TestSpeicherListe(t *testing.T) {
	s := neuerTestSpeicher(t)
	ctx := context.Background()

	if liste, err := s.Liste(ctx); err != nil || len(liste) != 0 {
		t.Fatalf("leere Liste erwartet, erhalten %v (Fehler %v)", liste, err)
	}

	for _, slug := range []string{"eins", "zwei", "drei"} {
		if err := s.Anlegen(ctx, testLink(slug)); err != nil {
			t.Fatalf("Anlegen %q: %v", slug, err)
		}
	}

	liste, err := s.Liste(ctx)
	if err != nil {
		t.Fatalf("Liste: %v", err)
	}
	if len(liste) != 3 {
		t.Fatalf("Liste: erwartet 3 Einträge, erhalten %d", len(liste))
	}
}

func TestSpeicherLoeschen(t *testing.T) {
	s := neuerTestSpeicher(t)
	ctx := context.Background()
	link := testLink("weg")

	if err := s.Anlegen(ctx, link); err != nil {
		t.Fatalf("Anlegen: %v", err)
	}
	if err := s.Loeschen(ctx, link.Slug); err != nil {
		t.Fatalf("Löschen: %v", err)
	}
	if _, err := s.Holen(ctx, link.Slug); !errors.Is(err, ErrNichtGefunden) {
		t.Fatalf("erwartet ErrNichtGefunden nach dem Löschen, erhalten %v", err)
	}
}

func TestSpeicherLoeschenUnbekannt(t *testing.T) {
	s := neuerTestSpeicher(t)

	if err := s.Loeschen(context.Background(), "gibtsnicht"); !errors.Is(err, ErrNichtGefunden) {
		t.Fatalf("erwartet ErrNichtGefunden, erhalten %v", err)
	}
}

func TestSpeicherAufrufZaehlen(t *testing.T) {
	s := neuerTestSpeicher(t)
	ctx := context.Background()
	link := testLink("zaehler")

	if err := s.Anlegen(ctx, link); err != nil {
		t.Fatalf("Anlegen: %v", err)
	}
	for i := 0; i < 3; i++ {
		if err := s.AufrufZaehlen(ctx, link.Slug); err != nil {
			t.Fatalf("AufrufZaehlen: %v", err)
		}
	}

	geholt, err := s.Holen(ctx, link.Slug)
	if err != nil {
		t.Fatalf("Holen: %v", err)
	}
	if geholt.Aufrufe != 3 {
		t.Errorf("Aufrufe: erwartet 3, erhalten %d", geholt.Aufrufe)
	}
}

func TestSpeicherAufrufZaehlenUnbekannt(t *testing.T) {
	s := neuerTestSpeicher(t)

	if err := s.AufrufZaehlen(context.Background(), "gibtsnicht"); !errors.Is(err, ErrNichtGefunden) {
		t.Fatalf("erwartet ErrNichtGefunden, erhalten %v", err)
	}
}

func TestSpeicherPruefenIstImmerBereit(t *testing.T) {
	s := neuerTestSpeicher(t)

	if err := s.Pruefen(context.Background()); err != nil {
		t.Fatalf("Pruefen: %v", err)
	}
}

// Der Speicher wird von allen Anfragen gleichzeitig benutzt. Mit -race fällt
// hier auf, wenn jemand eine Sperre vergisst; ohne diesen Test fiele es erst
// unter Last in Produktion auf.
func TestSpeicherVertraegtGleichzeitigenZugriff(t *testing.T) {
	s := neuerTestSpeicher(t)
	ctx := context.Background()
	link := testLink("parallel")

	if err := s.Anlegen(ctx, link); err != nil {
		t.Fatalf("Anlegen: %v", err)
	}

	const anzahl = 50
	var warten sync.WaitGroup
	warten.Add(anzahl * 2)
	for i := 0; i < anzahl; i++ {
		go func() {
			defer warten.Done()
			_ = s.AufrufZaehlen(ctx, link.Slug)
		}()
		go func() {
			defer warten.Done()
			_, _ = s.Liste(ctx)
		}()
	}
	warten.Wait()

	geholt, err := s.Holen(ctx, link.Slug)
	if err != nil {
		t.Fatalf("Holen: %v", err)
	}
	if geholt.Aufrufe != anzahl {
		t.Errorf("Aufrufe: erwartet %d, erhalten %d", anzahl, geholt.Aufrufe)
	}
}
