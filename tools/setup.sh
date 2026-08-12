#!/usr/bin/env bash
# Nach dem Klonen einmal ausführen. Aktiviert Hooks und Vorlagen.
set -euo pipefail
git config core.hooksPath .githooks
git config commit.template .gitmessage
chmod +x .githooks/* tools/*.sh deploy/scripts/*.sh
[ -f .git-verbotene-muster ] || printf '# Lokale Sperrliste, nicht versioniert.\n' > .git-verbotene-muster
echo "Fertig. Hooks sind aktiv."
