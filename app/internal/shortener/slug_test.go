package shortener

import (
	"errors"
	"strings"
	"testing"
)

func TestErzeugeSlugLaengeUndAlphabet(t *testing.T) {
	slug, err := ErzeugeSlug(7)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if len(slug) != 7 {
		t.Errorf("Laenge: erwartet 7, erhalten %d", len(slug))
	}
	for _, zeichen := range slug {
		if !strings.ContainsRune(alphabet, zeichen) {
			t.Errorf("Zeichen %q gehoert nicht zum Alphabet", zeichen)
		}
	}
}

func TestErzeugeSlugIstNichtKonstant(t *testing.T) {
	gesehen := make(map[string]bool)
	for i := 0; i < 50; i++ {
		slug, err := ErzeugeSlug(6)
		if err != nil {
			t.Fatalf("unerwarteter Fehler: %v", err)
		}
		gesehen[slug] = true
	}
	if len(gesehen) < 45 {
		t.Errorf("zu wenig Streuung: nur %d verschiedene Slugs aus 50", len(gesehen))
	}
}

func TestPruefeSlug(t *testing.T) {
	faelle := []struct {
		name     string
		eingabe  string
		erwartet error
	}{
		{"gueltig", "mein-link_1", nil},
		{"zu kurz", "ab", ErrSlugZuKurz},
		{"zu lang", strings.Repeat("a", 33), ErrSlugZuLang},
		{"Leerzeichen", "mein link", ErrSlugZeichen},
		{"Schraegstrich", "a/b/c", ErrSlugZeichen},
		{"Umlaut", "gruen", nil},
		{"reserviert", "api", ErrSlugReserviert},
		{"reserviert gross", "HEALTHZ", ErrSlugReserviert},
	}
	for _, fall := range faelle {
		t.Run(fall.name, func(t *testing.T) {
			err := PruefeSlug(fall.eingabe)
			if !errors.Is(err, fall.erwartet) {
				t.Errorf("erwartet %v, erhalten %v", fall.erwartet, err)
			}
		})
	}
}

func TestPruefeZiel(t *testing.T) {
	faelle := []struct {
		name     string
		eingabe  string
		erwartet error
	}{
		{"https", "https://www.beispiel.de/pfad?a=1", nil},
		{"http", "http://beispiel.de", nil},
		{"mit Leerzeichen aussen", "  https://beispiel.de  ", nil},
		{"leer", "", ErrZielLeer},
		{"nur Leerzeichen", "   ", ErrZielLeer},
		{"ohne Schema", "beispiel.de", ErrZielSchema},
		{"falsches Schema", "ftp://beispiel.de", ErrZielSchema},
		{"javascript", "javascript:alert(1)", ErrZielSchema},
		{"ohne Host", "https://", ErrZielOhneHost},
	}
	for _, fall := range faelle {
		t.Run(fall.name, func(t *testing.T) {
			_, err := PruefeZiel(fall.eingabe)
			if !errors.Is(err, fall.erwartet) {
				t.Errorf("erwartet %v, erhalten %v", fall.erwartet, err)
			}
		})
	}
}
