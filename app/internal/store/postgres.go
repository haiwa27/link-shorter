package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// pgSpeicher ist die PostgreSQL-Implementierung von Speicher (Story P-01).
//
// Sie ist der Grund, weshalb Blue/Green überhaupt trägt: beide Slots sprechen
// dieselbe Datenbank. Mit dem In-Memory-Speicher hielte jeder Slot eigene
// Daten, und ein Umschalten wäre aus Sicht der Nutzer ein Datenverlust.
type pgSpeicher struct {
	pool *pgxpool.Pool
}

// Schliessbar erfüllen Implementierungen, die Verbindungen halten. Der
// In-Memory-Speicher hält keine und implementiert die Schnittstelle nicht.
type Schliessbar interface {
	Schliessen()
}

// frist begrenzt jede einzelne Abfrage. Ohne sie könnte eine hängende
// Datenbankverbindung /healthz blockieren, und der Health-Check bliebe
// schlicht ohne Antwort statt ehrlich 503 zu melden.
const frist = 5 * time.Second

// pgUniqueVerletzung ist der SQLSTATE-Code für eine verletzte Eindeutigkeit.
const pgUniqueVerletzung = "23505"

// NeuerPostgresSpeicher legt den Verbindungspool an.
//
// Bewusst ohne Verbindungsaufbau: eine noch nicht erreichbare Datenbank darf
// den Start nicht verhindern. Der Prozess läuft weiter, /healthz meldet über
// Pruefen 503, und Caddy nimmt den Slot aus dem Verkehr. Das ist für die
// Verfügbarkeit besser als ein Container, der in einer Neustartschleife hängt.
func NeuerPostgresSpeicher(ctx context.Context, url string) (Speicher, error) {
	if url == "" {
		return nil, errors.New("Datenbank-URL ist leer")
	}
	konf, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("Datenbank-URL ist ungültig: %w", err)
	}
	// Zwei Slots teilen sich eine Datenbank. Der Pool bleibt klein, damit beide
	// zusammen die Verbindungsobergrenze von PostgreSQL nicht ausschöpfen.
	konf.MaxConns = 10
	konf.MinConns = 1
	konf.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, konf)
	if err != nil {
		return nil, fmt.Errorf("Verbindungspool konnte nicht angelegt werden: %w", err)
	}
	return &pgSpeicher{pool: pool}, nil
}

func (s *pgSpeicher) Schliessen() { s.pool.Close() }

func (s *pgSpeicher) Anlegen(ctx context.Context, link Link) error {
	ctx, abbrechen := context.WithTimeout(ctx, frist)
	defer abbrechen()

	_, err := s.pool.Exec(ctx,
		`INSERT INTO links (slug, ziel, erstellt, aufrufe) VALUES ($1, $2, $3, $4)`,
		link.Slug, link.Ziel, link.Erstellt, link.Aufrufe)
	if err != nil {
		var pgFehler *pgconn.PgError
		if errors.As(err, &pgFehler) && pgFehler.Code == pgUniqueVerletzung {
			return ErrBelegt
		}
		return fmt.Errorf("Anlegen von %q: %w", link.Slug, err)
	}
	return nil
}

func (s *pgSpeicher) Holen(ctx context.Context, slug string) (Link, error) {
	ctx, abbrechen := context.WithTimeout(ctx, frist)
	defer abbrechen()

	var link Link
	err := s.pool.QueryRow(ctx,
		`SELECT slug, ziel, erstellt, aufrufe FROM links WHERE slug = $1`, slug).
		Scan(&link.Slug, &link.Ziel, &link.Erstellt, &link.Aufrufe)
	if errors.Is(err, pgx.ErrNoRows) {
		return Link{}, ErrNichtGefunden
	}
	if err != nil {
		return Link{}, fmt.Errorf("Holen von %q: %w", slug, err)
	}
	return link, nil
}

func (s *pgSpeicher) Liste(ctx context.Context) ([]Link, error) {
	ctx, abbrechen := context.WithTimeout(ctx, frist)
	defer abbrechen()

	zeilen, err := s.pool.Query(ctx,
		`SELECT slug, ziel, erstellt, aufrufe FROM links ORDER BY erstellt DESC`)
	if err != nil {
		return nil, fmt.Errorf("Liste laden: %w", err)
	}
	defer zeilen.Close()

	liste := make([]Link, 0)
	for zeilen.Next() {
		var link Link
		if err := zeilen.Scan(&link.Slug, &link.Ziel, &link.Erstellt, &link.Aufrufe); err != nil {
			return nil, fmt.Errorf("Liste lesen: %w", err)
		}
		liste = append(liste, link)
	}
	if err := zeilen.Err(); err != nil {
		return nil, fmt.Errorf("Liste lesen: %w", err)
	}
	return liste, nil
}

func (s *pgSpeicher) Loeschen(ctx context.Context, slug string) error {
	ctx, abbrechen := context.WithTimeout(ctx, frist)
	defer abbrechen()

	ergebnis, err := s.pool.Exec(ctx, `DELETE FROM links WHERE slug = $1`, slug)
	if err != nil {
		return fmt.Errorf("Löschen von %q: %w", slug, err)
	}
	if ergebnis.RowsAffected() == 0 {
		return ErrNichtGefunden
	}
	return nil
}

// AufrufZaehlen erhöht den Zähler in der Datenbank statt im Prozess. Nur so
// zählen beide Slots auf denselben Wert; sonst hinge die Zahl davon ab, welcher
// Slot die Weiterleitung bedient hat.
func (s *pgSpeicher) AufrufZaehlen(ctx context.Context, slug string) error {
	ctx, abbrechen := context.WithTimeout(ctx, frist)
	defer abbrechen()

	ergebnis, err := s.pool.Exec(ctx,
		`UPDATE links SET aufrufe = aufrufe + 1 WHERE slug = $1`, slug)
	if err != nil {
		return fmt.Errorf("Aufruf zählen für %q: %w", slug, err)
	}
	if ergebnis.RowsAffected() == 0 {
		return ErrNichtGefunden
	}
	return nil
}

// Pruefen fragt die Datenbank wirklich. Ein Health-Check, der nur bestätigt,
// dass der Prozess lebt, würde einen Slot ohne Datenbank als bereit melden --
// und die Pipeline schaltete Verkehr auf eine Instanz, die keine Anfrage
// beantworten kann.
func (s *pgSpeicher) Pruefen(ctx context.Context) error {
	ctx, abbrechen := context.WithTimeout(ctx, frist)
	defer abbrechen()

	if err := s.pool.Ping(ctx); err != nil {
		return fmt.Errorf("Datenbank nicht erreichbar: %w", err)
	}
	return nil
}
