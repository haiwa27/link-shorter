// Paket shortener enthält die Fachlogik: Slugs erzeugen und Eingaben pruefen.
package shortener

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"strings"
)

// Ohne 0, O, I, l -- verwechslungsanfällige Zeichen weglassen, damit ein
// vorgelesener oder abgetippter Kurzlink funktioniert.
const alphabet = "abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"

const (
	SlugMinLaenge = 3
	SlugMaxLaenge = 32
)

var (
	ErrSlugZuKurz       = errors.New("Slug ist zu kurz")
	ErrSlugZuLang       = errors.New("Slug ist zu lang")
	ErrSlugZeichen      = errors.New("Slug enthält unerlaubte Zeichen")
	ErrSlugReserviert   = errors.New("Slug ist reserviert")
	ErrZielLeer         = errors.New("Ziel ist leer")
	ErrZielSchema       = errors.New("Ziel muss mit http:// oder https:// beginnen")
	ErrZielOhneHost     = errors.New("Ziel enthält keinen Hostnamen")
	ErrZielNichtParsbar = errors.New("Ziel ist keine gültige URL")
)

// reserviert schützt eigene Pfade davor, von einem Wunsch-Slug verdeckt zu
// werden (Story P-05).
var reserviert = map[string]bool{
	"api": true, "healthz": true, "metrics": true, "assets": true,
	"admin": true, "static": true, "favicon.ico": true,
}

// ErzeugeSlug liefert einen zufälligen Slug. crypto/rand statt math/rand,
// damit Kurzlinks nicht erratbar sind.
func ErzeugeSlug(laenge int) (string, error) {
	if laenge < SlugMinLaenge {
		return "", ErrSlugZuKurz
	}
	if laenge > SlugMaxLaenge {
		return "", ErrSlugZuLang
	}
	grenze := big.NewInt(int64(len(alphabet)))
	var b strings.Builder
	b.Grow(laenge)
	for i := 0; i < laenge; i++ {
		n, err := rand.Int(rand.Reader, grenze)
		if err != nil {
			return "", fmt.Errorf("Zufall nicht verfügbar: %w", err)
		}
		b.WriteByte(alphabet[n.Int64()])
	}
	return b.String(), nil
}

// PruefeSlug validiert einen vom Nutzer gewählten Slug.
func PruefeSlug(slug string) error {
	if len(slug) < SlugMinLaenge {
		return ErrSlugZuKurz
	}
	if len(slug) > SlugMaxLaenge {
		return ErrSlugZuLang
	}
	if reserviert[strings.ToLower(slug)] {
		return ErrSlugReserviert
	}
	for _, zeichen := range slug {
		erlaubt := (zeichen >= 'a' && zeichen <= 'z') ||
			(zeichen >= 'A' && zeichen <= 'Z') ||
			(zeichen >= '0' && zeichen <= '9') ||
			zeichen == '-' || zeichen == '_'
		if !erlaubt {
			return ErrSlugZeichen
		}
	}
	return nil
}

// PruefeZiel validiert die Ziel-URL und gibt sie normalisiert zurück.
func PruefeZiel(roh string) (string, error) {
	roh = strings.TrimSpace(roh)
	if roh == "" {
		return "", ErrZielLeer
	}
	zerlegt, err := url.Parse(roh)
	if err != nil {
		return "", ErrZielNichtParsbar
	}
	if zerlegt.Scheme != "http" && zerlegt.Scheme != "https" {
		return "", ErrZielSchema
	}
	if zerlegt.Host == "" {
		return "", ErrZielOhneHost
	}
	return zerlegt.String(), nil
}
