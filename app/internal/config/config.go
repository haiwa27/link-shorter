// Paket config liest die Konfiguration aus Umgebungsvariablen.
//
// Grundsatz: fehlende Pflichtwerte führen zum sofortigen Abbruch beim Start
// und nicht zu stillen Standardwerten, die in Produktion falsch sind.
package config

import (
	"fmt"
	"os"
	"strconv"
)

type Konfiguration struct {
	Adresse        string
	Umgebung       string
	Slot           string
	Version        string
	ChaosRate      float64
	WebVerzeichnis string
	DatenbankURL   string
}

func Laden() (Konfiguration, error) {
	cfg := Konfiguration{
		Adresse:        text("HEALTHGATE_ADRESSE", ":8080"),
		Umgebung:       text("HEALTHGATE_UMGEBUNG", "lokal"),
		Slot:           text("HEALTHGATE_SLOT", "lokal"),
		Version:        text("HEALTHGATE_VERSION", "dev"),
		WebVerzeichnis: text("HEALTHGATE_WEB_VERZEICHNIS", "/srv/web"),
		DatenbankURL:   os.Getenv("HEALTHGATE_DB_URL"),
	}

	rate, err := zahl("HEALTHGATE_CHAOS_RATE", 0)
	if err != nil {
		return cfg, err
	}
	if rate < 0 || rate > 1 {
		return cfg, fmt.Errorf("HEALTHGATE_CHAOS_RATE muss zwischen 0 und 1 liegen, ist %v", rate)
	}
	cfg.ChaosRate = rate

	// In Produktion sind Slot und Version Pflicht: ohne sie sind die Metriken
	// nicht zuordenbar und der Health Gate kann nicht auswerten.
	if cfg.Umgebung == "prod" {
		if cfg.Slot != "blue" && cfg.Slot != "green" {
			return cfg, fmt.Errorf("HEALTHGATE_SLOT muss in Produktion blue oder green sein, ist %q", cfg.Slot)
		}
		if cfg.Version == "" || cfg.Version == "dev" {
			return cfg, fmt.Errorf("HEALTHGATE_VERSION muss in Produktion gesetzt sein")
		}
		// Ohne gemeinsame Datenbank hielte jeder Slot eigene Daten. Das
		// Umschalten wäre dann aus Sicht der Nutzer ein Datenverlust, ohne dass
		// irgendein Health-Check darauf anspringt (Story P-01).
		if cfg.DatenbankURL == "" {
			return cfg, fmt.Errorf("HEALTHGATE_DB_URL muss in Produktion gesetzt sein")
		}
	}

	return cfg, nil
}

func text(schluessel, standard string) string {
	if wert := os.Getenv(schluessel); wert != "" {
		return wert
	}
	return standard
}

func zahl(schluessel string, standard float64) (float64, error) {
	wert := os.Getenv(schluessel)
	if wert == "" {
		return standard, nil
	}
	ergebnis, err := strconv.ParseFloat(wert, 64)
	if err != nil {
		return standard, fmt.Errorf("%s ist keine Zahl: %w", schluessel, err)
	}
	return ergebnis, nil
}
