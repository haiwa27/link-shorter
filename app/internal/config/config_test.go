package config

import "testing"

func TestLadenStandardwerte(t *testing.T) {
	t.Setenv("HEALTHGATE_UMGEBUNG", "lokal")

	cfg, err := Laden()
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if cfg.Adresse != ":8080" {
		t.Errorf("Adresse: erwartet :8080, erhalten %q", cfg.Adresse)
	}
	if cfg.ChaosRate != 0 {
		t.Errorf("ChaosRate: erwartet 0, erhalten %v", cfg.ChaosRate)
	}
}

func TestLadenProduktionOhneSlotSchlaegtFehl(t *testing.T) {
	t.Setenv("HEALTHGATE_UMGEBUNG", "prod")
	t.Setenv("HEALTHGATE_SLOT", "lokal")
	t.Setenv("HEALTHGATE_VERSION", "abc1234")

	if _, err := Laden(); err == nil {
		t.Fatal("erwartet: Fehler bei ungueltigem Slot in Produktion")
	}
}

func TestLadenUngueltigeChaosRate(t *testing.T) {
	t.Setenv("HEALTHGATE_CHAOS_RATE", "1.5")

	if _, err := Laden(); err == nil {
		t.Fatal("erwartet: Fehler bei ChaosRate ausserhalb 0 bis 1")
	}
}

func TestLadenProduktionOhneDatenbankSchlaegtFehl(t *testing.T) {
	t.Setenv("HEALTHGATE_UMGEBUNG", "prod")
	t.Setenv("HEALTHGATE_SLOT", "blue")
	t.Setenv("HEALTHGATE_VERSION", "abc1234")
	t.Setenv("HEALTHGATE_DB_URL", "")

	if _, err := Laden(); err == nil {
		t.Fatal("erwartet: Fehler ohne HEALTHGATE_DB_URL in Produktion")
	}
}

func TestLadenProduktionMitDatenbankIstGueltig(t *testing.T) {
	t.Setenv("HEALTHGATE_UMGEBUNG", "prod")
	t.Setenv("HEALTHGATE_SLOT", "green")
	t.Setenv("HEALTHGATE_VERSION", "abc1234")
	t.Setenv("HEALTHGATE_DB_URL", "postgres://nutzer:geheim@db:5432/healthgate?sslmode=disable")

	cfg, err := Laden()
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if cfg.Slot != "green" {
		t.Errorf("Slot: erwartet green, erhalten %q", cfg.Slot)
	}
}
