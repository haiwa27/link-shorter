// Paket store kapselt die Datenhaltung hinter einer Schnittstelle.
//
// Die Schnittstelle ist die Naht, an der später PostgreSQL eintritt, ohne dass
// Handler oder Tests angefasst werden müssen (Story P-01).
package store

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	ErrNichtGefunden = errors.New("link nicht gefunden")
	ErrBelegt        = errors.New("slug ist bereits belegt")
)

type Link struct {
	Slug     string    `json:"slug"`
	Ziel     string    `json:"ziel"`
	Erstellt time.Time `json:"erstellt"`
	Aufrufe  int64     `json:"aufrufe"`
}

type Speicher interface {
	Anlegen(ctx context.Context, link Link) error
	Holen(ctx context.Context, slug string) (Link, error)
	Liste(ctx context.Context) ([]Link, error)
	Loeschen(ctx context.Context, slug string) error
	AufrufZaehlen(ctx context.Context, slug string) error
	// Pruefen wird von /healthz aufgerufen und muss die Erreichbarkeit der
	// Datenhaltung nachweisen, nicht nur das Leben des Prozesses.
	Pruefen(ctx context.Context) error
}

// imSpeicher ist die Übergangsimplementierung für das Projektgerüst.
// TODO(P-01): durch eine PostgreSQL-Implementierung ersetzen. Solange dieser
// Speicher aktiv ist, funktioniert Blue/Green nur scheinbar: beide Slots halten
// getrennte Daten. Das ist genau der Punkt, an dem die gemeinsame Datenbank
// nötig wird.
type imSpeicher struct {
	mu    sync.RWMutex
	links map[string]Link
}

func NeuerSpeicher() Speicher {
	return &imSpeicher{links: make(map[string]Link)}
}

func (s *imSpeicher) Anlegen(_ context.Context, link Link) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, da := s.links[link.Slug]; da {
		return ErrBelegt
	}
	s.links[link.Slug] = link
	return nil
}

func (s *imSpeicher) Holen(_ context.Context, slug string) (Link, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	link, da := s.links[slug]
	if !da {
		return Link{}, ErrNichtGefunden
	}
	return link, nil
}

func (s *imSpeicher) Liste(_ context.Context) ([]Link, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	liste := make([]Link, 0, len(s.links))
	for _, link := range s.links {
		liste = append(liste, link)
	}
	return liste, nil
}

func (s *imSpeicher) Loeschen(_ context.Context, slug string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, da := s.links[slug]; !da {
		return ErrNichtGefunden
	}
	delete(s.links, slug)
	return nil
}

func (s *imSpeicher) AufrufZaehlen(_ context.Context, slug string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	link, da := s.links[slug]
	if !da {
		return ErrNichtGefunden
	}
	link.Aufrufe++
	s.links[slug] = link
	return nil
}

func (s *imSpeicher) Pruefen(_ context.Context) error {
	// TODO(O-01): sobald PostgreSQL angebunden ist, hier einen echten Ping
	// gegen die Datenbank ausführen und bei Fehler 503 melden.
	return nil
}
