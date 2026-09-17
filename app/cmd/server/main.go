// Kommando server startet die HTTP-Schnittstelle der Anwendung.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"healthgate/internal/chaos"
	"healthgate/internal/config"
	"healthgate/internal/handler"
	"healthgate/internal/metrics"
	"healthgate/internal/store"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.Laden()
	if err != nil {
		log.Error("Konfiguration ungültig", "fehler", err)
		os.Exit(1)
	}

	registry := metrics.NeueRegistry(cfg.Slot, cfg.Version)

	// Ohne Datenbank-URL bleibt der In-Memory-Speicher: das hält den lokalen
	// Start ohne Docker möglich. In Produktion erzwingt config.Laden die URL,
	// denn dort halten getrennte Daten je Slot das Blue/Green nur zum Schein
	// aufrecht (Story P-01).
	speicher := store.NeuerSpeicher()
	if cfg.DatenbankURL != "" {
		speicher, err = store.NeuerPostgresSpeicher(context.Background(), cfg.DatenbankURL)
		if err != nil {
			log.Error("Datenhaltung nicht verwendbar", "fehler", err)
			os.Exit(1)
		}
		log.Info("Datenhaltung: PostgreSQL")
	} else {
		log.Warn("Datenhaltung: im Prozess, Daten sind je Slot getrennt")
	}
	if s, ok := speicher.(store.Schliessbar); ok {
		defer s.Schliessen()
	}

	router := handler.NeuerRouter(handler.Abhaengigkeiten{
		Speicher:       speicher,
		Metriken:       registry,
		Version:        cfg.Version,
		Slot:           cfg.Slot,
		WebVerzeichnis: cfg.WebVerzeichnis,
		Log:            log,
	})

	// Reihenfolge der Middleware ist bewusst so gewählt: die Metrik-Schicht
	// liegt aussen und sieht daher auch die vom Chaos-Schalter erzeugten Fehler.
	// Genau diese Fehler sollen im Beobachtungsfenster sichtbar werden.
	kette := metrics.Middleware(registry, chaos.Middleware(cfg.ChaosRate, router))

	srv := &http.Server{
		Addr:              cfg.Adresse,
		Handler:           kette,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      15 * time.Second,
	}

	beenden := make(chan os.Signal, 1)
	signal.Notify(beenden, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Info("Server startet",
			"adresse", cfg.Adresse,
			"slot", cfg.Slot,
			"version", cfg.Version,
			"chaosRate", cfg.ChaosRate)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("Server abgebrochen", "fehler", err)
			os.Exit(1)
		}
	}()

	<-beenden
	log.Info("Server wird beendet")

	// Auslaufzeit, damit der Reverse Proxy laufende Anfragen noch zu Ende
	// bringen kann. Ohne das entstehen beim Umschalten kurze 5xx (Story R-02).
	ctx, abbrechen := context.WithTimeout(context.Background(), 10*time.Second)
	defer abbrechen()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("Beenden nicht sauber", "fehler", err)
	}
}
