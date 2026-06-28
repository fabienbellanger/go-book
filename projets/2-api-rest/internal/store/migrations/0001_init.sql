-- Migration initiale : table des tâches.
-- done est stocké en INTEGER (0/1) et created_at en époque Unix (secondes),
-- pour rester portable d'un driver à l'autre (pas de type DATE/BOOL spécifique).
CREATE TABLE IF NOT EXISTS tasks (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    title      TEXT    NOT NULL,
    done       INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_tasks_done ON tasks (done);
