PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    pseudo TEXT UNIQUE NOT NULL,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at DATETIME NOT NULL
);

INSERT OR IGNORE INTO users(id, pseudo, email, password_hash, created_at)
VALUES (1, 'DemoUser', 'demo@example.com', 'hashed', datetime('now'));

CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    expires_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS rooms (
    id TEXT PRIMARY KEY,
    code TEXT NOT NULL,
    host_user_id INTEGER REFERENCES users(id),
    game_type TEXT NOT NULL CHECK (game_type IN ('waiting', 'in_progress', 'finished')),
    status TEXT NOT NULL CHECK (status IN ('waiting', 'in_progress', 'finished')),
    created_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS room_players (
    room_id TEXT REFERENCES rooms(id) ON DELETE CASCADE,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    joined_at DATETIME NOT NULL,
    PRIMARY KEY (room_id, user_id)
);

CREATE TABLE IF NOT EXISTS games (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    room_id TEXT REFERENCES rooms(id) ON DELETE CASCADE,
    config_json TEXT NOT NULL,
    max_rounds INTEGER NOT NULL,
    current_round INTEGER NOT NULL DEFAULT 0,
    started_at DATETIME,
    ended_at DATETIME
);

CREATE TABLE IF NOT EXISTS rounds (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    game_id INTEGER REFERENCES games(id) ON DELETE CASCADE,
    round_index INTEGER NOT NULL,
    payload_json TEXT NOT NULL,
    started_at DATETIME,
    ended_at DATETIME
);

CREATE TABLE IF NOT EXISTS answers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    round_id INTEGER REFERENCES rounds(id) ON DELETE CASCADE,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    data_json TEXT NOT NULL,
    submitted_at DATETIME NOT NULL,
    is_valid INTEGER NOT NULL CHECK (is_valid IN (0, 1)) DEFAULT NULL
);

CREATE TABLE IF NOT EXISTS validations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    answer_id INTEGER REFERENCES answers(id) ON DELETE CASCADE,
    voter_user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    vote INTEGER NOT NULL CHECK (vote IN (0, 1))
);

CREATE TABLE IF NOT EXISTS scores (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    game_id INTEGER REFERENCES games(id) ON DELETE CASCADE,
    points INTEGER NOT NULL DEFAULT 0
);
