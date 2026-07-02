-- Migration 0002 : table d'archive.
-- Archiver une tâche, c'est la retirer de « tasks » ET l'insérer ici — les deux
-- écritures doivent être atomiques (tout ou rien). Voir SQLStore.ArchiveTask,
-- qui les enveloppe dans une transaction database/sql.
-- archived_at est l'époque Unix (secondes) du moment de l'archivage.
CREATE TABLE IF NOT EXISTS tasks_archive (
    id          INTEGER PRIMARY KEY,
    title       TEXT    NOT NULL,
    done        INTEGER NOT NULL,
    created_at  INTEGER NOT NULL,
    archived_at INTEGER NOT NULL
);
