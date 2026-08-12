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
