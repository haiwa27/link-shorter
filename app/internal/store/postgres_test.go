package store

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Die Tests gegen eine echte Datenbank laufen nur, wenn HEALTHGATE_TEST_DB_URL
// gesetzt ist. Die Pipeline startet dafür eine Wegwerf-Instanz; lokal ohne
// Datenbank werden sie übersprungen statt rot zu melden.
const testURLSchluessel = "HEALTHGATE_TEST_DB_URL"

// migrationsDatei ist dieselbe Datei, die docker-entrypoint-initdb.d in
// Staging und Produktion anwendet. Zwei Schemadefinitionen, die auseinander
// laufen können, wären die teuerste Art, sich selbst zu täuschen.
const migrationsDatei = "../../migrations/0001_init.sql"

func testSpeicher(t *testing.T) *pgSpeicher {
	t.Helper()

	url := os.Getenv(testURLSchluessel)
	if url == "" {
		t.Skipf("%s nicht gesetzt, Datenbanktests werden übersprungen", testURLSchluessel)
	}

	ctx := context.Background()
	speicher, err := NeuerPostgresSpeicher(ctx, url)
	if err != nil {
		t.Fatalf("Speicher anlegen: %v", err)
	}
	pg, ok := speicher.(*pgSpeicher)
	if !ok {
		t.Fatalf("unerwarteter Typ %T", speicher)
	}
	t.Cleanup(pg.Schliessen)

	schema, err := os.ReadFile(migrationsDatei)
	if err != nil {
		t.Fatalf("Migration lesen: %v", err)
	}
	if _, err := pg.pool.Exec(ctx, string(schema)); err != nil {
		t.Fatalf("Migration anwenden: %v", err)
	}
	if _, err := pg.pool.Exec(ctx, "TRUNCATE links"); err != nil {
		t.Fatalf("Tabelle leeren: %v", err)
	}
	return pg
}

func beispielLink(slug string) Link {
	return Link{
		Slug:     slug,
		Ziel:     "https://beispiel.de/" + slug,
		Erstellt: time.Now().UTC().Truncate(time.Millisecond),
	}
}

func TestNeuerPostgresSpeicherLehntLeereURLAb(t *testing.T) {
	if _, err := NeuerPostgresSpeicher(context.Background(), ""); err == nil {
		t.Fatal("erwartet: Fehler bei leerer Datenbank-URL")
	}
}

func TestNeuerPostgresSpeicherLehntUngueltigeURLAb(t *testing.T) {
	if _, err := NeuerPostgresSpeicher(context.Background(), "kein-gueltiges-ziel://"); err == nil {
		t.Fatal("erwartet: Fehler bei ungültiger Datenbank-URL")
	}
}

func TestPostgresAnlegenUndHolen(t *testing.T) {
	pg := testSpeicher(t)
	ctx := context.Background()
	link := beispielLink("abc1234")

	if err := pg.Anlegen(ctx, link); err != nil {
		t.Fatalf("Anlegen: %v", err)
	}

	geholt, err := pg.Holen(ctx, link.Slug)
	if err != nil {
		t.Fatalf("Holen: %v", err)
	}
	if geholt.Ziel != link.Ziel {
		t.Errorf("Ziel: erwartet %q, erhalten %q", link.Ziel, geholt.Ziel)
	}
	if geholt.Aufrufe != 0 {
		t.Errorf("Aufrufe: erwartet 0, erhalten %d", geholt.Aufrufe)
	}
	if !geholt.Erstellt.Equal(link.Erstellt) {
		t.Errorf("Erstellt: erwartet %v, erhalten %v", link.Erstellt, geholt.Erstellt)
	}
}

func TestPostgresDoppelterSlugErgibtBelegt(t *testing.T) {
	pg := testSpeicher(t)
	ctx := context.Background()
	link := beispielLink("doppelt")

	if err := pg.Anlegen(ctx, link); err != nil {
		t.Fatalf("Anlegen: %v", err)
	}
	err := pg.Anlegen(ctx, link)
	if !errors.Is(err, ErrBelegt) {
		t.Fatalf("erwartet ErrBelegt, erhalten %v", err)
	}
}

func TestPostgresHolenUnbekannt(t *testing.T) {
	pg := testSpeicher(t)

	_, err := pg.Holen(context.Background(), "gibtsnicht")
	if !errors.Is(err, ErrNichtGefunden) {
		t.Fatalf("erwartet ErrNichtGefunden, erhalten %v", err)
	}
}

func TestPostgresListeLiefertAlleEintraege(t *testing.T) {
	pg := testSpeicher(t)
	ctx := context.Background()

	for _, slug := range []string{"eins", "zwei", "drei"} {
		if err := pg.Anlegen(ctx, beispielLink(slug)); err != nil {
			t.Fatalf("Anlegen %q: %v", slug, err)
		}
	}

	liste, err := pg.Liste(ctx)
	if err != nil {
		t.Fatalf("Liste: %v", err)
	}
	if len(liste) != 3 {
		t.Fatalf("Liste: erwartet 3 Einträge, erhalten %d", len(liste))
	}
}

func TestPostgresListeIstLeerOhneEintraege(t *testing.T) {
	pg := testSpeicher(t)

	liste, err := pg.Liste(context.Background())
	if err != nil {
		t.Fatalf("Liste: %v", err)
	}
	if len(liste) != 0 {
		t.Fatalf("Liste: erwartet 0 Einträge, erhalten %d", len(liste))
	}
}

func TestPostgresAufrufZaehlen(t *testing.T) {
	pg := testSpeicher(t)
	ctx := context.Background()
	link := beispielLink("zaehler")

	if err := pg.Anlegen(ctx, link); err != nil {
		t.Fatalf("Anlegen: %v", err)
	}
	for i := 0; i < 3; i++ {
		if err := pg.AufrufZaehlen(ctx, link.Slug); err != nil {
			t.Fatalf("AufrufZaehlen: %v", err)
		}
	}

	geholt, err := pg.Holen(ctx, link.Slug)
	if err != nil {
		t.Fatalf("Holen: %v", err)
	}
	if geholt.Aufrufe != 3 {
		t.Errorf("Aufrufe: erwartet 3, erhalten %d", geholt.Aufrufe)
	}
}

func TestPostgresAufrufZaehlenUnbekannt(t *testing.T) {
	pg := testSpeicher(t)

	err := pg.AufrufZaehlen(context.Background(), "gibtsnicht")
	if !errors.Is(err, ErrNichtGefunden) {
		t.Fatalf("erwartet ErrNichtGefunden, erhalten %v", err)
	}
}

func TestPostgresLoeschen(t *testing.T) {
	pg := testSpeicher(t)
	ctx := context.Background()
	link := beispielLink("weg")

	if err := pg.Anlegen(ctx, link); err != nil {
		t.Fatalf("Anlegen: %v", err)
	}
	if err := pg.Loeschen(ctx, link.Slug); err != nil {
		t.Fatalf("Löschen: %v", err)
	}
	if _, err := pg.Holen(ctx, link.Slug); !errors.Is(err, ErrNichtGefunden) {
		t.Fatalf("erwartet ErrNichtGefunden nach dem Löschen, erhalten %v", err)
	}
}

func TestPostgresLoeschenUnbekannt(t *testing.T) {
	pg := testSpeicher(t)

	err := pg.Loeschen(context.Background(), "gibtsnicht")
	if !errors.Is(err, ErrNichtGefunden) {
		t.Fatalf("erwartet ErrNichtGefunden, erhalten %v", err)
	}
}

func TestPostgresPruefenMeldetErreichbarkeit(t *testing.T) {
	pg := testSpeicher(t)

	if err := pg.Pruefen(context.Background()); err != nil {
		t.Fatalf("Pruefen: %v", err)
	}
}

// Der Nachweis, dass Pruefen wirklich die Datenbank fragt und nicht nur nil
// zurückgibt: gegen einen geschlossenen Pool muss es fehlschlagen.
func TestPostgresPruefenSchlaegtOhneVerbindungFehl(t *testing.T) {
	url := os.Getenv(testURLSchluessel)
	if url == "" {
		t.Skipf("%s nicht gesetzt, Datenbanktests werden übersprungen", testURLSchluessel)
	}

	speicher, err := NeuerPostgresSpeicher(context.Background(), url)
	if err != nil {
		t.Fatalf("Speicher anlegen: %v", err)
	}
	pg, ok := speicher.(*pgSpeicher)
	if !ok {
		t.Fatalf("unerwarteter Typ %T", speicher)
	}
	if err := pg.Pruefen(context.Background()); err != nil {
		t.Fatalf("Pruefen vor dem Schliessen: %v", err)
	}

	pg.Schliessen()
	if err := pg.Pruefen(context.Background()); err == nil {
		t.Fatal("erwartet: Fehler nach dem Schliessen des Pools")
	}
}

// Der Pool muss auch unter gleichzeitigem Zugriff tragen: beide Slots bedienen
// Anfragen parallel, und der Zähler wird in der Datenbank erhöht, nicht im
// Prozess.
func TestPostgresGleichzeitigesZaehlen(t *testing.T) {
	pg := testSpeicher(t)
	ctx := context.Background()
	link := beispielLink("parallel")

	if err := pg.Anlegen(ctx, link); err != nil {
		t.Fatalf("Anlegen: %v", err)
	}

	const anzahl = 20
	fehler := make(chan error, anzahl)
	for i := 0; i < anzahl; i++ {
		go func() { fehler <- pg.AufrufZaehlen(ctx, link.Slug) }()
	}
	for i := 0; i < anzahl; i++ {
		if err := <-fehler; err != nil {
			t.Fatalf("AufrufZaehlen: %v", err)
		}
	}

	geholt, err := pg.Holen(ctx, link.Slug)
	if err != nil {
		t.Fatalf("Holen: %v", err)
	}
	if geholt.Aufrufe != anzahl {
		t.Errorf("Aufrufe: erwartet %d, erhalten %d", anzahl, geholt.Aufrufe)
	}
}

// Die Poolgrenze ist kein Schmuck: zwei Slots teilen sich eine Datenbank.
// Der Test belegt, dass die gesetzte Obergrenze im Pool ankommt.
func TestPostgresPoolGrenzeWirdGesetzt(t *testing.T) {
	url := os.Getenv(testURLSchluessel)
	if url == "" {
		t.Skipf("%s nicht gesetzt, Datenbanktests werden übersprungen", testURLSchluessel)
	}

	if _, err := pgxpool.ParseConfig(url); err != nil {
		t.Fatalf("URL zerlegen: %v", err)
	}

	speicher, err := NeuerPostgresSpeicher(context.Background(), url)
	if err != nil {
		t.Fatalf("Speicher anlegen: %v", err)
	}
	pg := speicher.(*pgSpeicher)
	t.Cleanup(pg.Schliessen)

	if pg.pool.Config().MaxConns != 10 {
		t.Errorf("MaxConns: erwartet 10, erhalten %d", pg.pool.Config().MaxConns)
	}
}

// Sicherstellen, dass die Implementierung die Schnittstellen erfüllt. Ein
// Compilerfehler hier ist besser als ein Laufzeitfehler in Produktion.
var _ Speicher = (*pgSpeicher)(nil)
var _ Schliessbar = (*pgSpeicher)(nil)
