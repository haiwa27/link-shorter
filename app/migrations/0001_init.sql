-- Migration 0001: Grundtabelle fuer Kurzlinks (Story P-01)
--
-- Anwenden vorlaeufig per psql. TODO(P-01): Migrationswerkzeug festlegen
-- (z.B. golang-migrate) und in die Pipeline einhaengen.

CREATE TABLE IF NOT EXISTS links (
    slug        TEXT PRIMARY KEY,
    ziel        TEXT        NOT NULL,
    erstellt    TIMESTAMPTZ NOT NULL DEFAULT now(),
    aufrufe     BIGINT      NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS links_erstellt_idx ON links (erstellt DESC);
